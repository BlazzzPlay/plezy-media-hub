# Plezy Upstream Integration Contract

## Purpose

Keep Plezy aligned with official releases without losing the custom Media Operations workflow.

## Branch roles

- `upstream/main`: read-only official repository reference.
- `feature/media-operations-mvp`: current production customization branch.
- `sync/plezy-<version>`: disposable integration branch created from an official tag.
- `backup/plezy-media-operations-<version>`: immutable tag created before each integration.

Never merge upstream directly into the production branch.

## Protected custom surface

These paths are product-owned and must be restored and validated in every sync:

- `lib/models/media_operations/`
- `lib/services/media_operations/`
- `lib/screens/manage/manage_screen.dart`
- `services/media-orchestrator/`
- `docs/media-operations-mvp.md`

## Integration seams

These files connect the custom feature to Plezy and are the only expected conflict points:

- `lib/screens/main_screen.dart`
- `lib/widgets/side_navigation_rail.dart`
- `android/app/build.gradle.kts`
- `android/app/src/main/AndroidManifest.xml`
- `pubspec.yaml`

Keep edits in these files minimal and documented in the sync commit.

## Update procedure

1. Fetch `upstream` and select an official tag.
2. Confirm the production worktree is clean.
3. Create and push a `backup/...` tag.
4. Create `sync/plezy-<version>` from the official tag in a separate worktree.
5. Port protected custom paths first, then adapt each integration seam.
6. Set version to `<official>-<custom-revision>+<build>`.
7. Run verification before merging into the production branch:
   - `flutter analyze`
   - Media Operations service tests
   - Android release build on the VPS
   - Manual checks: menu entry, connection, candidates, plans, queue, health, APK update detection
8. Commit the validated integration branch, merge it into the production branch, then publish the APK.

## Rollback

If any validation fails, abandon the `sync/...` branch and continue running the existing production branch. The `backup/...` tag is the immutable recovery point.