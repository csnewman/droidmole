use log::{debug, error, info};
use nix::fcntl::OFlag;
use nix::unistd;
use nix::unistd::{setsid, ForkResult, Pid};
use std::ffi::CString;
use std::os::fd::RawFd;
use std::time::Duration;
use std::{panic, thread};

mod assets;
mod logger;
mod tracer;

fn main() {
    logger::configure();

    info!("DroidMole Init Injector");

    // TODO: Replace with better error handling
    panic::set_hook(Box::new(|info| loop {
        error!("PANIC: {}", info);
        thread::sleep(Duration::from_secs(1));
    }));

    // Use pipe to synchronise fork
    let (parker, unparker) =
        unistd::pipe2(OFlag::O_CLOEXEC | OFlag::O_DIRECT).expect("failed to create pipe");

    // init process must run with pid=1
    // Fork and run injection logic in another process
    debug!("Forking");
    unsafe {
        match unistd::fork().expect("failed to fork") {
            ForkResult::Parent { .. } => {
                unistd::close(unparker).expect("failed to close");
                handle_parent(parker);
            }
            ForkResult::Child => {
                unistd::close(parker).expect("failed to close");
                handle_child(unparker);
            }
        }
    }
}

fn handle_parent(parker: RawFd) {
    info!("Waiting for ptrace");

    let mut dest = [0];
    unistd::read(parker, &mut dest).expect("failed to read");
    assert_eq!(dest[0], 123);

    info!("Spawning original init process");
    let process_path = CString::new("/original-init").unwrap();
    unistd::execve::<&CString, &CString>(&process_path, &[&process_path], &[])
        .expect("failed to start original-init");
}

fn handle_child(unparker: RawFd) {
    let parent_id = Pid::parent();
    setsid().expect("failed to setsid");

    tracer::trace(parent_id, unparker);
}
