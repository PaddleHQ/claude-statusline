# Symbols font

`symbols.woff2` is a subset of Symbols Nerd Font Mono (`SymbolsNerdFontMono-Regular.ttf`), from `NerdFontsSymbolsOnly.zip` in the [Nerd Fonts v3.5.1 release](https://github.com/ryanoasis/nerd-fonts/releases/tag/v3.5.1). The screenshot generator embeds it in an image only when the image draws a Nerd Font icon, because GitHub shows the images through `<img>`, where the reader's own fonts cannot be relied upon.

The subset holds the icons the status line draws with `icons: nerd`, and the progress capsule glyphs U+EE00 to U+EE05. It was made with fontTools 4.68.0:

```sh
pyftsubset SymbolsNerdFontMono-Regular.ttf --flavor=woff2 --output-file=tools/screenshots/symbols.woff2 \
  --unicodes=U+E0A0,U+EE00-EE05,U+F013,U+F017,U+F019,U+F071,U+F073,U+F07B,U+F093,U+F0E7,U+F135,U+F1BB,U+F035B,U+F06A9
```

To add an icon, append its code point to `--unicodes`, run the command again on the same release's font, and run `go run ./tools/screenshots`.

## Licence

The font is distributed under the MIT licence, which follows in full.

```text
The MIT License (MIT)

Copyright (c) 2014 Ryan L McIntyre

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
```
