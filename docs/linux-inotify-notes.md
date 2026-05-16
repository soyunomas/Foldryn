# Linux inotify notes

Foldryn 0.3.0 listens primarily for:

- `IN_CLOSE_WRITE`: a file was closed after writing.
- `IN_MOVED_TO`: a file appeared by rename, typical for completed browser downloads.
- `IN_CREATE`: a file or directory was created.

It deliberately ignores temporary download suffixes before calling `stat`. This prevents normal browser rename races from being logged as errors.

For many recursive folders, Linux may require a higher watch limit:

```bash
cat /proc/sys/fs/inotify/max_user_watches
sudo sysctl fs.inotify.max_user_watches=524288
```
