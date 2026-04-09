# Archived Services

This directory contains services that have been archived due to architectural changes or feature deprecation.

## go-approval

**Archived Date:** 2026-04-07  
**Reason:** Phase 2 architectural change - approval workflow moved to brand's own systems

### Background

According to `docs/archive/B端重新设计了一些功能.md` §4, the system-internal approval workflow has been removed. Brands now handle approvals in their own ERP/OA systems and directly call RCProtocol's execution APIs using API Key authentication.

### What was implemented

- Complete approval workflow service (~3200 lines of code)
- Approval CRUD operations
- Approval/rejection logic with role-based access control
- Downstream action execution (calling rc-api after approval)
- Database schema: `approvals` table in `rcprotocol_approval` database
- Full test coverage

### Impact of removal

- go-approval service is no longer needed
- Brand publishing, policy application, and other operations no longer require approval context (`X-Approval-Id`)
- Brands use API Key authentication to directly call rc-api execution endpoints
- go-workorder service has been refactored to remove approval dependency (Spec-11 Phase 2)

### How to restore (if needed in the future)

1. Move `services/archive/go-approval/` back to `services/go-approval/`
2. Uncomment go-approval service in `deploy/compose/docker-compose.yml`
3. Uncomment approval routes in `services/go-gateway/internal/proxy/router.go`
4. Restore `deploy/postgres/init/006_create_approval_db.sql` if deleted
5. Update dependent services to use approval context again

### Related documentation

- Original spec: `.kiro/specs/spec-10-go-approval/`
- Refactoring plan: `docs/refactoring-plan.md` Phase 2
- Architecture change: `docs/archive/B端重新设计了一些功能.md` §4
