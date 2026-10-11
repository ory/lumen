use serde::{Deserialize, Serialize};
use time::OffsetDateTime;

#[derive(Debug, Serialize)]
pub struct SessionResponse {
    pub session_id: String,
    pub user_id: String,
    #[serde(with = "time::serde::rfc3339")]
    pub expires_at: OffsetDateTime,
}

#[derive(Debug, Deserialize)]
pub struct LoginRequest {
    pub email: String,
    pub password: String,
}

pub fn session_response(session: &crate::store::Session) -> SessionResponse {
    SessionResponse {
        session_id: session.id.to_string(),
        user_id: session.user_id.to_string(),
        expires_at: session.expires_at,
    }
}
