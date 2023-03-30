use anyhow::Result;
use log::info;
use nix::errno::Errno;
use nix::libc;
use nix::libc::user_regs_struct;
use nix::unistd::Pid;
use std::ffi::c_void;
use std::mem::{transmute, MaybeUninit};
use std::ptr;

use nix::sys::ptrace::RequestType;
pub use nix::sys::ptrace::{
    detach, getevent, read, setoptions, syscall, AddressType, Event, Options,
};
use nix::sys::signal::Signal;
use nix::sys::wait;
use nix::sys::wait::WaitStatus;

pub const PTRACE_SEIZE: RequestType = 0x4206;

pub fn seize(pid: Pid, options: Options) -> Result<()> {
    unsafe {
        return ptrace_other(
            PTRACE_SEIZE as RequestType,
            pid,
            ptr::null_mut(),
            options.bits() as *mut c_void,
        );
    }
}

unsafe fn ptrace_other(
    request: RequestType,
    pid: Pid,
    addr: AddressType,
    data: *mut c_void,
) -> Result<()> {
    Errno::result(libc::ptrace(
        request as RequestType,
        libc::pid_t::from(pid),
        addr,
        data,
    ))?;

    return Ok(());
}

pub fn getregs(pid: Pid) -> Result<user_regs_struct> {
    ptrace_get_data::<user_regs_struct>(libc::PTRACE_GETREGS, pid)
}

pub fn setregs(pid: Pid, regs: user_regs_struct) -> Result<()> {
    let res = unsafe {
        libc::ptrace(
            libc::PTRACE_SETREGS as RequestType,
            libc::pid_t::from(pid),
            ptr::null_mut::<c_void>(),
            &regs as *const _ as *const c_void,
        )
    };
    Errno::result(res)?;

    return Ok(());
}

fn ptrace_get_data<T>(request: RequestType, pid: Pid) -> Result<T> {
    let mut data = MaybeUninit::uninit();
    let res = unsafe {
        libc::ptrace(
            request,
            libc::pid_t::from(pid),
            ptr::null_mut::<T>(),
            data.as_mut_ptr() as *const _ as *const c_void,
        )
    };
    Errno::result(res)?;
    Ok(unsafe { data.assume_init() })
}

fn read_string(pid: Pid, mut addr: u64) -> String {
    let mut result = String::new();
    'outer: loop {
        let mut got = read(pid, addr as AddressType).expect("failed to read");

        for _ in 0..8 {
            let c = got & 0xFF;
            if c == 0 {
                break 'outer;
            }

            result.push(c as u8 as char);
            got >>= 8;
        }

        addr += 8;
    }

    return result;
}

pub enum TracerEvent {
    PTraceEvent(Pid, Signal, Event),
    PTraceSyscall(Pid),
    Stopped(Pid, Signal),
}

pub fn wait_for_event(pid: Pid) -> Result<TracerEvent> {
    loop {
        let status = match wait::waitpid(Some(pid), Some(wait::WaitPidFlag::__WALL)) {
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
                panic!("Continued not implemented");
            }
            WaitStatus::StillAlive => {
                panic!("StillAlive not implemented");
            }
        }
    }
}
