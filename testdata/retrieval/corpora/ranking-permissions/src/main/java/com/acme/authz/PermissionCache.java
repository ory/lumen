package com.acme.authz;

import java.util.Map;
import java.util.Set;
import java.util.concurrent.ConcurrentHashMap;
import java.util.concurrent.atomic.AtomicLong;

/** In-memory cache of resolved permission grants per user. */
public final class PermissionCache {
  private final PolicyStore policies;
  private final Map<String, Set<String>> grants = new ConcurrentHashMap<>();
  private final AtomicLong evictions = new AtomicLong();

  public PermissionCache(PolicyStore policies) {
    this.policies = policies;
  }

  private Set<String> grantsFor(String userId) {
    return grants.computeIfAbsent(userId, policies::resolve);
  }

  /** Reports whether the user may read project resources. */
  public boolean canReadProject(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("project:read") || granted.contains("project:admin");
  }

  /** Reports whether the user may write project resources. */
  public boolean canWriteProject(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("project:write") || granted.contains("project:admin");
  }

  /** Reports whether the user may read repository resources. */
  public boolean canReadRepository(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("repository:read") || granted.contains("repository:admin");
  }

  /** Reports whether the user may write repository resources. */
  public boolean canWriteRepository(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("repository:write") || granted.contains("repository:admin");
  }

  /** Reports whether the user may read issue resources. */
  public boolean canReadIssue(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("issue:read") || granted.contains("issue:admin");
  }

  /** Reports whether the user may write issue resources. */
  public boolean canWriteIssue(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("issue:write") || granted.contains("issue:admin");
  }

  /** Reports whether the user may read comment resources. */
  public boolean canReadComment(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("comment:read") || granted.contains("comment:admin");
  }

  /** Reports whether the user may write comment resources. */
  public boolean canWriteComment(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("comment:write") || granted.contains("comment:admin");
  }

  /** Reports whether the user may read release resources. */
  public boolean canReadRelease(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("release:read") || granted.contains("release:admin");
  }

  /** Reports whether the user may write release resources. */
  public boolean canWriteRelease(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("release:write") || granted.contains("release:admin");
  }

  /** Reports whether the user may read pipeline resources. */
  public boolean canReadPipeline(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("pipeline:read") || granted.contains("pipeline:admin");
  }

  /** Reports whether the user may write pipeline resources. */
  public boolean canWritePipeline(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("pipeline:write") || granted.contains("pipeline:admin");
  }

  /** Reports whether the user may read secret resources. */
  public boolean canReadSecret(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("secret:read") || granted.contains("secret:admin");
  }

  /** Reports whether the user may write secret resources. */
  public boolean canWriteSecret(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("secret:write") || granted.contains("secret:admin");
  }

  /** Reports whether the user may read runner resources. */
  public boolean canReadRunner(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("runner:read") || granted.contains("runner:admin");
  }

  /** Reports whether the user may write runner resources. */
  public boolean canWriteRunner(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("runner:write") || granted.contains("runner:admin");
  }

  /** Reports whether the user may read webhook resources. */
  public boolean canReadWebhook(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("webhook:read") || granted.contains("webhook:admin");
  }

  /** Reports whether the user may write webhook resources. */
  public boolean canWriteWebhook(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("webhook:write") || granted.contains("webhook:admin");
  }

  /** Reports whether the user may read package resources. */
  public boolean canReadPackage(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("package:read") || granted.contains("package:admin");
  }

  /** Reports whether the user may write package resources. */
  public boolean canWritePackage(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("package:write") || granted.contains("package:admin");
  }

  /** Reports whether the user may read wiki resources. */
  public boolean canReadWiki(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("wiki:read") || granted.contains("wiki:admin");
  }

  /** Reports whether the user may write wiki resources. */
  public boolean canWriteWiki(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("wiki:write") || granted.contains("wiki:admin");
  }

  /** Reports whether the user may read milestone resources. */
  public boolean canReadMilestone(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("milestone:read") || granted.contains("milestone:admin");
  }

  /** Reports whether the user may write milestone resources. */
  public boolean canWriteMilestone(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("milestone:write") || granted.contains("milestone:admin");
  }

  /** Reports whether the user may read label resources. */
  public boolean canReadLabel(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("label:read") || granted.contains("label:admin");
  }

  /** Reports whether the user may write label resources. */
  public boolean canWriteLabel(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("label:write") || granted.contains("label:admin");
  }

  /** Reports whether the user may read environment resources. */
  public boolean canReadEnvironment(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("environment:read") || granted.contains("environment:admin");
  }

  /** Reports whether the user may write environment resources. */
  public boolean canWriteEnvironment(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("environment:write") || granted.contains("environment:admin");
  }

  /** Reports whether the user may read deployment resources. */
  public boolean canReadDeployment(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("deployment:read") || granted.contains("deployment:admin");
  }

  /** Reports whether the user may write deployment resources. */
  public boolean canWriteDeployment(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("deployment:write") || granted.contains("deployment:admin");
  }

  /** Drops every cached grant for the user so the next check reloads from the policy store. */
  public void invalidatePermissions(String userId) {
    grants.remove(userId);
    evictions.incrementAndGet();
  }

  /** Reports whether the user may read artifact resources. */
  public boolean canReadArtifact(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("artifact:read") || granted.contains("artifact:admin");
  }

  /** Reports whether the user may write artifact resources. */
  public boolean canWriteArtifact(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("artifact:write") || granted.contains("artifact:admin");
  }

  /** Reports whether the user may read snippet resources. */
  public boolean canReadSnippet(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("snippet:read") || granted.contains("snippet:admin");
  }

  /** Reports whether the user may write snippet resources. */
  public boolean canWriteSnippet(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("snippet:write") || granted.contains("snippet:admin");
  }

  /** Reports whether the user may read board resources. */
  public boolean canReadBoard(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("board:read") || granted.contains("board:admin");
  }

  /** Reports whether the user may write board resources. */
  public boolean canWriteBoard(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("board:write") || granted.contains("board:admin");
  }

  /** Reports whether the user may read epic resources. */
  public boolean canReadEpic(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("epic:read") || granted.contains("epic:admin");
  }

  /** Reports whether the user may write epic resources. */
  public boolean canWriteEpic(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("epic:write") || granted.contains("epic:admin");
  }

  /** Reports whether the user may read team resources. */
  public boolean canReadTeam(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("team:read") || granted.contains("team:admin");
  }

  /** Reports whether the user may write team resources. */
  public boolean canWriteTeam(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("team:write") || granted.contains("team:admin");
  }

  /** Reports whether the user may read member resources. */
  public boolean canReadMember(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("member:read") || granted.contains("member:admin");
  }

  /** Reports whether the user may write member resources. */
  public boolean canWriteMember(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("member:write") || granted.contains("member:admin");
  }

  /** Reports whether the user may read invite resources. */
  public boolean canReadInvite(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("invite:read") || granted.contains("invite:admin");
  }

  /** Reports whether the user may write invite resources. */
  public boolean canWriteInvite(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("invite:write") || granted.contains("invite:admin");
  }

  /** Reports whether the user may read token resources. */
  public boolean canReadToken(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("token:read") || granted.contains("token:admin");
  }

  /** Reports whether the user may write token resources. */
  public boolean canWriteToken(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("token:write") || granted.contains("token:admin");
  }

  /** Reports whether the user may read auditLog resources. */
  public boolean canReadAuditLog(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("auditLog:read") || granted.contains("auditLog:admin");
  }

  /** Reports whether the user may write auditLog resources. */
  public boolean canWriteAuditLog(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("auditLog:write") || granted.contains("auditLog:admin");
  }

  /** Reports whether the user may read billing resources. */
  public boolean canReadBilling(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("billing:read") || granted.contains("billing:admin");
  }

  /** Reports whether the user may write billing resources. */
  public boolean canWriteBilling(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("billing:write") || granted.contains("billing:admin");
  }

  /** Reports whether the user may read domain resources. */
  public boolean canReadDomain(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("domain:read") || granted.contains("domain:admin");
  }

  /** Reports whether the user may write domain resources. */
  public boolean canWriteDomain(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("domain:write") || granted.contains("domain:admin");
  }

  /** Reports whether the user may read certificate resources. */
  public boolean canReadCertificate(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("certificate:read") || granted.contains("certificate:admin");
  }

  /** Reports whether the user may write certificate resources. */
  public boolean canWriteCertificate(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("certificate:write") || granted.contains("certificate:admin");
  }

  /** Reports whether the user may read branch resources. */
  public boolean canReadBranch(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("branch:read") || granted.contains("branch:admin");
  }

  /** Reports whether the user may write branch resources. */
  public boolean canWriteBranch(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("branch:write") || granted.contains("branch:admin");
  }

  /** Reports whether the user may read tag resources. */
  public boolean canReadTag(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("tag:read") || granted.contains("tag:admin");
  }

  /** Reports whether the user may write tag resources. */
  public boolean canWriteTag(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("tag:write") || granted.contains("tag:admin");
  }

  /** Reports whether the user may read mergeRequest resources. */
  public boolean canReadMergeRequest(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("mergeRequest:read") || granted.contains("mergeRequest:admin");
  }

  /** Reports whether the user may write mergeRequest resources. */
  public boolean canWriteMergeRequest(String userId) {
    Set<String> granted = grantsFor(userId);
    return granted.contains("mergeRequest:write") || granted.contains("mergeRequest:admin");
  }

  public long evictionCount() {
    return evictions.get();
  }
}
