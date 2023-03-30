use crate::assets::DEVICE_AGENT;
use anyhow::Result;
use log::{debug, info, warn};
use nix::sys::ptrace;
use nix::sys::ptrace::Event;
use nix::sys::signal::Signal;
use nix::sys::wait::{waitpid, WaitPidFlag, WaitStatus};
use nix::unistd;
use nix::unistd::{chroot, ForkResult, Pid};
use std::ffi::CString;
use std::fs;
use std::io::Write;
use std::mem::transmute;
use std::os::fd::RawFd;
use std::os::unix::fs::OpenOptionsExt;
use std::path::Path;

pub fn trace(init_pid: Pid, unparker: RawFd) {
    info!("Seizing {}", init_pid);
    ptrace::seize(
        init_pid,
        ptrace::Options::PTRACE_O_TRACEEXEC |
            ptrace::Options::PTRACE_O_TRACECLONE |
            // TODO: track
            // ptrace::Options::PTRACE_O_EXITKILL |
            // ptrace::Options::PTRACE_O_TRACEEXIT |
            ptrace::Options::PTRACE_O_TRACEFORK |
            ptrace::Options::PTRACE_O_TRACEVFORK |
            ptrace::Options::PTRACE_O_TRACESYSGOOD,
    )
    .expect("failed to ptrace process");

    info!("Unparking init process");

    let dest = [123];
    unistd::write(unparker, &dest).expect("failed to unpark");

    info!("Watching init process");

    let mut found_proc = false;
    let mut should_detach = Vec::new();

    'outer: loop {
        let tevent = wait_for_event().expect("failed to wait for event");

        match tevent {
            TracerEvent::PTraceEvent(pid, _, evt) => {
                if should_detach.contains(&pid) {
                    should_detach.retain(|&x| x != pid);

                    info!("Detaching");
                    match ptrace::detach(pid, None) {
                        Ok(_) => {}
                        Err(err) => {
                            info!("Ignoring {}", err)
                        }
                    }
                    continue 'outer;
                }

                match evt {
                    Event::PTRACE_EVENT_FORK => {
                        let got =
                            Pid::from_raw(ptrace::getevent(pid).expect("failed to get pid") as i32);
                        debug!("fork {:?}=>{:?}", pid, got);

                        // No need to track deeper than 1 level
                        if pid != init_pid {
                            info!("Trying to detach {}", got);
                            match ptrace::detach(got, None) {
                                Ok(_) => {}
                                Err(err) => {
                                    debug!("Ignoring {}", err);
                                    debug!("Marking for late detach");
                                    should_detach.push(got);
                                }
                            }
                        }
                    }
                    Event::PTRACE_EVENT_VFORK => {
                        unimplemented!("vfork not implemented");
                    }
                    Event::PTRACE_EVENT_CLONE => {
                        let got =
                            Pid::from_raw(ptrace::getevent(pid).expect("failed to get pid") as i32);
                        debug!("clone {:?}=>{:?}", pid, got);

                        // No need to track deeper than 1 level
                        if pid != init_pid {
                            info!("Trying to detach {}", got);
                            match ptrace::detach(got, None) {
                                Ok(_) => {}
                                Err(err) => {
                                    debug!("Ignoring {}", err);
                                    debug!("Marking for late detach");
                                    should_detach.push(got);
                                }
                            }
                        }
                    }
                    Event::PTRACE_EVENT_EXEC => {
                        debug!("ptrace exec {}", pid);

                        if pid != init_pid && found_proc {
                            // Find which binary is being exec'd
                            // cwd is /proc
                            let base = Path::new("./").join(pid.to_string());
                            let proc_exe = base.join("exe");
                            let proc_cmd = base.join("cmdline");

                            let exe_link = proc_exe.read_link().unwrap_or_default();
                            let exe_path = exe_link.to_str().unwrap_or_default();

                            // Extract arguments
                            let args;
                            match fs::read_to_string(proc_cmd) {
                                Ok(d) => {
                                    args = d
                                        .split('\x00')
                                        .map(|s| s.to_string())
                                        .filter(|s| s.len() > 0)
                                        .collect::<Vec<String>>();
                                }
                                Err(_) => {
                                    args = Vec::new();
                                }
                            }

                            debug!("Exec pid={} exe={} args={:?}", pid, exe_path, args);

                            // Check if exec-ing to zygote process
                            if exe_path == "/system/bin/app_process64"
                                && args.len() > 1
                                && args[1] == "-Xzygote"
                            {
                                info!("Found zygote");

                                // Wait till zygote starts loading
                                match ptrace::syscall(pid, None) {
                                    Ok(_) => {}
                                    Err(err) => {
                                        info!("Ignoring {}", err)
                                    }
                                }
                                let _ = waitpid(pid, None).expect("failed to wait");
                                // TODO: Validate status

                                // Pause zygote process so we can transfer ptrace ownership
                                info!("Pausing zygote");
                                ptrace::detach(pid, Some(Signal::SIGSTOP))
                                    .expect("expected to stop");

                                debug!("Forking");
                                unsafe {
                                    match unistd::fork().expect("failed to fork") {
                                        ForkResult::Parent { .. } => {
                                            // TODO: Watch for device-agent crashes
                                        }
                                        ForkResult::Child => {
                                            spawn_device_agent(pid);
                                        }
                                    }
                                }
                            } else {
                                info!("Detaching");
                                ptrace::detach(pid, None).expect("expected to stop");
                            }

                            continue 'outer;
                        }
                    }
                    Event::PTRACE_EVENT_VFORK_DONE => {
                        unimplemented!("vfork_done");
                    }
                    Event::PTRACE_EVENT_EXIT => {
                        warn!("Exit event process not implemented {}", pid);
                    }
                    Event::PTRACE_EVENT_SECCOMP => {
                        unimplemented!("seccomp");
                    }
                    Event::PTRACE_EVENT_STOP => {
                        info!("Stop event process not implemented {}", pid);
                    }
                    _ => {
                        panic!("Unknown event {}", evt as i32);
                    }
                }

                match match found_proc {
                    true => ptrace::cont(pid, None),
                    false => ptrace::syscall(pid, None),
                } {
                    Ok(_) => {}
                    Err(err) => {
                        info!("Ignoring {}", err)
                    }
                }
            }
            TracerEvent::PTraceSyscall(pid) => {
                // init process will configure /proc before switching the system root. Change
                // working dir to /proc to stay inside root filesystem when the root switches.
                if !found_proc {
                    if let Ok(paths) = fs::read_dir("/proc/") {
                        if paths.count() > 0 {
                            info!("Found /proc");
                            unistd::chdir("/proc").expect("Failed to change dir");
                            found_proc = true;
                        }
                    }
                }

                if should_detach.contains(&pid) {
                    should_detach.retain(|&x| x != pid);

                    info!("Trying to detach {}", pid);
                    match ptrace::detach(pid, None) {
                        Ok(_) => {}
                        Err(err) => {
                            debug!("Ignoring {}", err)
                        }
                    }
                    continue 'outer;
                }

                match match found_proc {
                    true => ptrace::cont(pid, None),
                    false => ptrace::syscall(pid, None),
                } {
                    Ok(_) => {}
                    Err(err) => {
                        info!("Ignoring {}", err)
                    }
                }
            }
            TracerEvent::Stopped(pid, sig) => {
                if should_detach.contains(&pid) {
                    should_detach.retain(|&x| x != pid);

                    info!("Trying to detach {}", pid);
                    match ptrace::detach(pid, sig) {
                        Ok(_) => {}
                        Err(err) => {
                            debug!("Ignoring {}", err)
                        }
                    }
                    continue 'outer;
                }

                match match found_proc {
                    true => ptrace::cont(pid, sig),
                    false => ptrace::syscall(pid, sig),
                } {
                    Ok(_) => {}
                    Err(err) => {
                        info!("Ignoring {}", err)
                    }
                }
            }
        }
    }
}

fn spawn_device_agent(zygote_pid: Pid) {
    debug!("Changing root to system root");
    chroot("../").expect("failed to chroot");

    info!("Unpacking device-agent");
    fs::OpenOptions::new()
        .create(true)
        .write(true)
        .mode(0o777)
        .open("/data/local/tmp/device-agent")
        .unwrap()
        .write_all(DEVICE_AGENT)
        .expect("failed to unpack device-agent");

    info!("Spawning device-agent");
    let process_path = CString::new("/data/local/tmp/device-agent").unwrap();
    let arg1 = CString::new(zygote_pid.as_raw().to_string()).unwrap();

    unistd::execve::<&CString, &CString>(&process_path, &[&process_path, &arg1], &[])
        .expect("failed to start device-agent");
    unreachable!()
}

enum TracerEvent {
    PTraceEvent(Pid, Signal, Event),
    PTraceSyscall(Pid),
    Stopped(Pid, Signal),
}

fn wait_for_event() -> Result<TracerEvent> {
    loop {
        let status = match waitpid(
            None,
            Some(WaitPidFlag::__WALL | WaitPidFlag::WCONTINUED | WaitPidFlag::WUNTRACED),
        ) {
            Ok(ok) => ok,
            Err(err) => {
                info!("Failed to wait for event {}", err);
                continue;
            }
        };

        match status {
            WaitStatus::Exited(_, _) => {
                // TODO
            }
            WaitStatus::Signaled(pid, sig, other) => {
                unimplemented!("Signaled not implemented {} {} {}", pid, sig, other);
            }
            WaitStatus::Stopped(pid, sig) => {
                return Ok(TracerEvent::Stopped(pid, sig));
            }
            WaitStatus::PtraceEvent(pid, sig, other) => {
                let evt: Event = unsafe { transmute(other) };
                return Ok(TracerEvent::PTraceEvent(pid, sig, evt));
            }
            WaitStatus::PtraceSyscall(pid) => {
                return Ok(TracerEvent::PTraceSyscall(pid));
            }
            WaitStatus::Continued(_) => {
                unimplemented!("Continued not implemented");
            }
            WaitStatus::StillAlive => {
                unimplemented!("StillAlive not implemented");
            }
        }
    }
}
