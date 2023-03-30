mod logger;
mod ptrace;

use crate::ptrace::{wait_for_event, TracerEvent};
use frida::{DeviceManager, Frida};
use log::{debug, error, info};
use nix::libc;
use nix::libc::{
    c_long, c_ulonglong, SYS_epoll_pwait, SYS_epoll_wait, SYS_epoll_wait_old, SYS_poll, SYS_ppoll,
    SYS_pselect6, SYS_select, EINTR,
};
use nix::sys::signal;
use nix::sys::signal::Signal;
use nix::sys::wait::waitpid;
use nix::unistd::Pid;
use std::env::args;
use std::time::Duration;
use std::{panic, thread};

fn main() {
    logger::configure();

    // TODO: Replace with better error handling
    panic::set_hook(Box::new(|info| loop {
        error!("PANIC: {}", info);
        thread::sleep(Duration::from_secs(1));
    }));

    info!("DroidMole Device Agent");

    let args: Vec<_> = args().collect();
    let zygote_pid_raw = args
        .get(1)
        .expect("pid argument missing")
        .parse::<u64>()
        .expect("invalid pid");

    info!("Zygote pid: {}", zygote_pid_raw);

    let zygote_pid = Pid::from_raw(zygote_pid_raw as libc::pid_t);

    let frida = unsafe { Frida::obtain() };
    frida.set_log_handler(|d, l, m| {
        info!("[frida] domain={} level={} msg={}", d, l, m);
    });

    frida.execute(move || {
        info!("Seizing zygote");
        ptrace::seize(zygote_pid, ptrace::Options::PTRACE_O_TRACESYSGOOD)
            .expect("failed to seize zygote");

        // TODO: Validate result
        let result = waitpid(zygote_pid, None).expect("failed to wait");
        debug!("Wait result: {:?}", result);

        info!("Resuming zygote");
        signal::kill(zygote_pid, Signal::SIGCONT).expect("failed to send cont");

        info!("Waiting for zygote to hit injectable state");
        ptrace::syscall(zygote_pid, None).expect("failed to restart");

        // Wait until zygote hits poll instruction, which is a reliable place to inject at
        'outer: loop {
            let tevent = wait_for_event(zygote_pid).expect("failed to wait for event");

            match tevent {
                TracerEvent::PTraceEvent(pid, _, _) => {
                    ptrace::syscall(pid, None).expect("failed to resume");
                }
                TracerEvent::PTraceSyscall(pid) => {
                    let mut regs = ptrace::getregs(pid).expect("failed to get regs");

                    // TODO: Support non-x86_64 systems
                    let id = regs.orig_rax as c_long;

                    // TODO: Investigate why hooking after SYS_setpgid fails
                    if id == SYS_select
                        || id == SYS_pselect6
                        || id == SYS_poll
                        || id == SYS_ppoll
                        || id == SYS_epoll_wait
                        || id == SYS_epoll_wait_old
                        || id == SYS_epoll_pwait
                    {
                        info!("Zygote {} reached injectable syscall {}", zygote_pid, id);

                        // Fake syscall
                        regs.orig_rax = -1i64 as u64;
                        ptrace::setregs(pid, regs).expect("failed to set regs before");
                        ptrace::syscall(pid, None).expect("failed to restart");

                        // TODO: Validate result
                        let result = waitpid(pid, None).expect("failed to wait");
                        debug!("Wait result: {:?}", result);

                        // Fake result
                        let mut regs = ptrace::getregs(pid).expect("failed to get regs");
                        regs.rax = (-EINTR) as c_ulonglong;
                        ptrace::setregs(zygote_pid, regs).expect("failed to set regs");
                        break 'outer;
                    } else {
                        ptrace::syscall(pid, None).expect("failed to restart");
                    }
                }
                TracerEvent::Stopped(pid, sig) => {
                    ptrace::syscall(pid, sig).expect("failed to restart");
                }
            }
        }

        info!("Zygote ready");
    });

    // Get "local" frida device
    let device_manager = DeviceManager::obtain(&frida);
    let device = device_manager
        .get_device_by_type()
        .expect("failed to get local");

    info!("Frida Device {} ({})", device.get_id(), device.get_name());

    // Handle "gated" children
    device.on_child_added(|child| {
        info!("Child added!");
        let pid = child.get_pid();
        info!("child-pid={}", pid);
        let _child_session = device.attach(pid).expect("failed to attach to child");
        // TODO: Configure child_session
        device.resume(pid).expect("failed to resume child");
    });

    // Inject frida into zygote
    info!("Attaching to Zygote");
    let session = device
        .attach(zygote_pid_raw as u32)
        .expect("failed to attach?");
    session
        .enable_child_gating()
        .expect("failed to enable child gating");

    // Resume Zygote
    info!("Injection complete");
    info!("Resuming Zygote");
    frida.execute(move || {
        ptrace::detach(zygote_pid, None).expect("failed to detach");
    });

    loop {
        info!("Injection active");
        // TODO: Add logic
        thread::sleep(Duration::from_millis(1000));
    }
}
