//! Durable storage for delivery receipts.

use std::fs::{self, File, OpenOptions};
use std::io;
use std::path::{Path, PathBuf};

/// A pool of open file handles for the receipt log.
#[derive(Debug)]
pub struct StoragePool {
    root: PathBuf,
    log: File,
}

/// Opens (creating if necessary) the receipt store rooted at `path` and
/// returns a pool ready for appends.
pub fn connect(path: &Path) -> io::Result<StoragePool> {
    fs::create_dir_all(path)?;
    let log = OpenOptions::new()
        .create(true)
        .append(true)
        .open(path.join("receipts.log"))?;
    Ok(StoragePool {
        root: path.to_path_buf(),
        log,
    })
}

impl StoragePool {
    /// Directory holding the receipt log.
    pub fn root(&self) -> &Path {
        &self.root
    }
}
