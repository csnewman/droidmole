use log::{Level, Metadata, Record};

struct InitLogger;

static LOGGER: InitLogger = InitLogger;

pub fn configure() {
    log::set_max_level(log::LevelFilter::Debug);
    log::set_logger(&LOGGER).expect("failed to set logger");
}

impl log::Log for InitLogger {
    fn enabled(&self, metadata: &Metadata) -> bool {
        metadata.level() <= Level::Info
    }

    fn log(&self, record: &Record) {
        if self.enabled(record.metadata()) {
            println!(
                "[{}][{}] {}",
                record.module_path().unwrap_or("unknown"),
                record.level(),
                record.args()
            );
        }
    }

    fn flush(&self) {}
}
