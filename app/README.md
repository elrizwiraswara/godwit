# Godwit desktop app

The Flutter user interface for Godwit. It runs the bundled engine on this computer, or connects to a Godwit server. See the [main README](../README.md) for what Godwit does.

## Development

From the repository root, build the engine into `app/backend_dist` once:

```powershell
powershell -ExecutionPolicy Bypass -File scripts\build_windows.ps1 -SkipInstaller
```

Then, in this folder:

- `flutter run -d windows` runs the app.
- `flutter test` runs the unit tests, the phone-size layout tests and the engine tests.
- `flutter test integration_test -d windows` drives the UI with the real engine.

The app icon comes from `assets/brand/godwit-icon.svg` in the repository root; after changing it, run `python tools/brand/render_icons.py` from the root.
