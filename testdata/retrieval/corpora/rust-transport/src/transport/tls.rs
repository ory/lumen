use std::io;
use std::net::SocketAddr;
use std::sync::Arc;
use std::time::Duration;

use rustls::pki_types::ServerName;
use rustls::{ClientConfig, ClientConnection, StreamOwned};

use super::tcp::{self, TcpConnection};

/// A TLS session layered over a [`TcpConnection`].
pub struct TlsConnection {
    session: StreamOwned<ClientConnection, std::net::TcpStream>,
}

/// Establishes a TLS session with `server_name` by first opening a TCP
/// connection and then running the rustls handshake on top of it.
pub fn connect(
    addr: &SocketAddr,
    server_name: ServerName<'static>,
    config: Arc<ClientConfig>,
    timeout: Duration,
) -> io::Result<TlsConnection> {
    let tcp: TcpConnection = tcp::connect(addr, timeout)?;
    let conn = ClientConnection::new(config, server_name)
        .map_err(|err| io::Error::new(io::ErrorKind::InvalidData, err))?;
    let socket = tcp.stream().try_clone()?;
    Ok(TlsConnection {
        session: StreamOwned::new(conn, socket),
    })
}
