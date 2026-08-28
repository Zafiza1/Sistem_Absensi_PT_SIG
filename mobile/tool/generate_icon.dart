// Composes the launcher-icon source images from the raw brand mark
// (assets/images/logo.png). Run once whenever the logo changes:
//   dart run tool/generate_icon.dart
//   dart run flutter_launcher_icons
//
// The raw logo has a transparent background and a wide (non-square)
// aspect ratio, which flutter_launcher_icons would otherwise stretch or
// crop unpredictably across launchers. This script pre-composes two
// proper square sources instead:
//   - icon_foreground.png: logo centered in the safe zone of a
//     transparent 1024x1024 canvas, for Android adaptive icons (paired
//     with a solid white adaptive_icon_background in pubspec.yaml).
//   - icon_flat.png: logo centered on a solid white 1024x1024 canvas,
//     for the plain (non-adaptive) launcher icon.
import 'dart:io';

import 'package:image/image.dart' as img;

void main() {
  final logoBytes = File('assets/images/logo.png').readAsBytesSync();
  final decoded = img.decodePng(logoBytes);
  if (decoded == null) {
    stderr.writeln('Could not decode assets/images/logo.png');
    exitCode = 1;
    return;
  }
  // The source file has transparent padding baked in around the mark
  // itself (it isn't tightly cropped), which threw off centering when
  // composing onto a new square canvas. Trim to the actual opaque content
  // first so every composed variant centers the mark itself, not its
  // original canvas.
  final logo = img.trim(decoded, mode: img.TrimMode.transparent);

  const canvasSize = 1024;

  File('assets/images/icon_foreground.png').writeAsBytesSync(
    img.encodePng(_compose(logo, canvasSize: canvasSize, fitFactor: 0.5, background: null)),
  );
  File('assets/images/icon_flat.png').writeAsBytesSync(
    img.encodePng(_compose(logo, canvasSize: canvasSize, fitFactor: 0.62, background: img.ColorRgba8(255, 255, 255, 255))),
  );

  stdout.writeln('Generated icon_foreground.png and icon_flat.png');
}

img.Image _compose(img.Image logo, {required int canvasSize, required double fitFactor, img.Color? background}) {
  final canvas = img.Image(width: canvasSize, height: canvasSize, numChannels: 4);
  img.fill(canvas, color: background ?? img.ColorRgba8(0, 0, 0, 0));

  final maxDim = canvasSize * fitFactor;
  final scale = maxDim / logo.width < maxDim / logo.height ? maxDim / logo.width : maxDim / logo.height;
  final resized = img.copyResize(
    logo,
    width: (logo.width * scale).round(),
    height: (logo.height * scale).round(),
    interpolation: img.Interpolation.cubic,
  );

  final offsetX = (canvasSize - resized.width) ~/ 2;
  final offsetY = (canvasSize - resized.height) ~/ 2;
  img.compositeImage(canvas, resized, dstX: offsetX, dstY: offsetY);
  return canvas;
}
