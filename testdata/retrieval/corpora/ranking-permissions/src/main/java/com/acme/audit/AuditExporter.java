package com.acme.audit;

import java.io.Writer;
import java.io.IOException;

/** Streams audit events to the SIEM as CSV. */
public final class AuditExporter {
  public void export(Iterable<AuditEvent> events, Writer out) throws IOException {
    for (AuditEvent e : events) {
      String category = switch (e.action()) {
        case "invalidatePermissions", "grantPermissions", "revokePermissions" -> "authz";
        case "login", "logout" -> "session";
        default -> "other";
      };
      out.write(e.at() + "," + category + "," + e.action() + "," + e.userId() + "\n");
    }
  }
}
