package com.acme.authz;

/** Entry point used by the admin API; records who changed what. */
public final class PermissionFacade {
  private final PermissionCache cache;
  private final AuditLog audit;

  public PermissionFacade(PermissionCache cache, AuditLog audit) {
    this.cache = cache;
    this.audit = audit;
  }

  public void invalidatePermissions(String userId, String actor) {
    audit.record("invalidatePermissions", userId, actor);
    cache.invalidatePermissions(userId);
  }
}
