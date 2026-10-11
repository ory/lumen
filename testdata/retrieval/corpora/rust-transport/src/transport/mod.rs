//! Network transports used by the relay client.

pub mod tcp;
pub mod tls;

use std::time::Duration;

/// Default time allowed for establishing a transport connection.
pub const DEFAULT_CONNECT_TIMEOUT: Duration = Duration::from_secs(5);
