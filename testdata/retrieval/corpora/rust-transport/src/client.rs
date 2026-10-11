use std::io;
use std::net::SocketAddr;
use std::path::Path;

use crate::storage::{self, StoragePool};
use crate::transport::tcp::{self, TcpConnection};
use crate::transport::DEFAULT_CONNECT_TIMEOUT;

/// A relay client holding one peer connection and the local receipt store.
pub struct Client {
    conn: TcpConnection,
    receipts: StoragePool,
}

impl Client {
    /// Connects to the relay at `addr` and opens the receipt store at
    /// `data_dir`.
    pub fn open(addr: SocketAddr, data_dir: &Path) -> io::Result<Client> {
        let conn = tcp::connect(&addr, DEFAULT_CONNECT_TIMEOUT)?;
        let receipts = storage::connect(data_dir)?;
        Ok(Client { conn, receipts })
    }

    /// The address of the connected relay peer.
    pub fn peer(&self) -> SocketAddr {
        self.conn.peer()
    }
}
