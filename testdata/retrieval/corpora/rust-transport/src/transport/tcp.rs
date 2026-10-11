use std::io;
use std::net::{SocketAddr, TcpStream};
use std::time::Duration;

/// A plain TCP connection to a relay peer.
#[derive(Debug)]
pub struct TcpConnection {
    stream: TcpStream,
    peer: SocketAddr,
}

impl TcpConnection {
    /// The remote address this connection is bound to.
    pub fn peer(&self) -> SocketAddr {
        self.peer
    }

    /// Borrow the underlying socket.
    pub fn stream(&self) -> &TcpStream {
        &self.stream
    }
}

/// Opens a TCP socket to `addr`, failing if the handshake does not finish
/// within `timeout`. Nagle's algorithm is disabled because relay frames are
/// small and latency sensitive.
pub fn connect(addr: &SocketAddr, timeout: Duration) -> io::Result<TcpConnection> {
    let stream = TcpStream::connect_timeout(addr, timeout)?;
    stream.set_nodelay(true)?;
    stream.set_read_timeout(Some(timeout))?;
    Ok(TcpConnection {
        stream,
        peer: *addr,
    })
}
