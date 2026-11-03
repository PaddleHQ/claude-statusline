# Symbols font

`symbols.woff` is a subset of Symbols Nerd Font Mono (`SymbolsNerdFontMono-Regular.ttf`), from `NerdFontsSymbolsOnly.zip` in the [Nerd Fonts v3.5.1 release](https://github.com/ryanoasis/nerd-fonts/releases/tag/v3.5.1). The screenshot generator embeds it in an image only when the image draws a Nerd Font icon, because GitHub shows the images through `<img>`, where the reader's own fonts cannot be relied upon.

The subset holds the icons the status line draws with `icons: nerd`, and the progress capsule glyphs U+EE00 to U+EE05. It is WOFF rather than WOFF2 so that `TestSymbolsFontCoversEveryIcon` can read it with the standard library, which has zlib but no Brotli. It was made with fontTools 4.68.0:

```sh
pyftsubset SymbolsNerdFontMono-Regular.ttf --flavor=woff --output-file=tools/screenshots/symbols.woff \
  --unicodes=U+E0A0,U+EE00-EE05,U+F013,U+F017,U+F019,U+F071,U+F073,U+F07B,U+F093,U+F0E7,U+F135,U+F1BB,U+F035B,U+F06A9
```

To add an icon, append its code point to `--unicodes`, run the command again on the same release's font, and run `go run ./tools/screenshots`. `TestSymbolsFontCoversEveryIcon` fails while a Private Use Area code point in the Go code is missing from the subset.

## Licences

Symbols Nerd Font Mono gathers glyphs from several icon sets, each under its own licence. The table below says where each glyph in the subset comes from, as given by the `patch_set` ranges in the release's [`font-patcher`](https://github.com/ryanoasis/nerd-fonts/blob/v3.5.1/font-patcher) and the icon set table in the release's README and [`src/glyphs/README.md`](https://github.com/ryanoasis/nerd-fonts/blob/v3.5.1/src/glyphs/README.md). The full licence texts are in [`licences/`](licences), copied unchanged from the files named below except for trailing whitespace.

| Glyphs | Icon set (range in the font) | Upstream | Licence | Text |
|---|---|---|---|---|
| U+E0A0 | Powerline Symbols (U+E0A0 to U+E0A2) | [powerline/powerline](https://github.com/powerline/powerline) | MIT ([`src/glyphs/powerline-symbols/LICENSE.txt`](https://github.com/ryanoasis/nerd-fonts/blob/v3.5.1/src/glyphs/powerline-symbols/LICENSE.txt)) | [`powerline-symbols.txt`](licences/powerline-symbols.txt) |
| U+EE00 to U+EE05 | Progress Indicators, from `extraglyphs.sfd` (U+EE00 to U+EE0B) | [source-foundry/Hack](https://github.com/source-foundry/Hack), as the README lists for `extraglyphs` | MIT, as the README lists. Hack's own [`LICENSE.md`](https://github.com/source-foundry/Hack/blob/v3.003/LICENSE.md) adds the Bitstream Vera License for the Vera glyphs it derives from. | [`hack.txt`](licences/hack.txt) |
| U+F013, U+F017, U+F019, U+F071, U+F073, U+F07B, U+F093, U+F0E7, U+F135, U+F1BB | Font Awesome 6.5.1 (U+ED00 to U+F2FF) | [FortAwesome/Font-Awesome](https://github.com/FortAwesome/Font-Awesome) | Icons CC BY 4.0, fonts SIL OFL 1.1 ([`src/glyphs/font-awesome/LICENSE.txt`](https://github.com/ryanoasis/nerd-fonts/blob/v3.5.1/src/glyphs/font-awesome/LICENSE.txt)). Nerd Fonts' [licence audit](https://github.com/ryanoasis/nerd-fonts/blob/v3.5.1/license-audit.md) says it uses the icons, under CC BY 4.0. | [`font-awesome.txt`](licences/font-awesome.txt) |
| U+F035B, U+F06A9 | Material Design Icons (U+F0001 to U+F1AF0) | [Templarian/MaterialDesign-Font](https://github.com/Templarian/MaterialDesign-Font) | Apache 2.0 ([`src/glyphs/materialdesign/LICENSE`](https://github.com/ryanoasis/nerd-fonts/blob/v3.5.1/src/glyphs/materialdesign/LICENSE)) | [`material-design-icons.txt`](licences/material-design-icons.txt), [`apache-2.0.txt`](licences/apache-2.0.txt) |

The font as a whole is Nerd Fonts' work. The release zip ships the MIT licence below. The repository's [`LICENSE`](https://github.com/ryanoasis/nerd-fonts/blob/v3.5.1/LICENSE) puts patched fonts under the SIL OFL 1.1, copyright (c) 2014 Ryan L McIntyre, and its source files under the MIT licence; it is copied in full in [`nerd-fonts.txt`](licences/nerd-fonts.txt).

Notices for the glyphs in the subset:

- Font Awesome Free 6.5.1 by Fonticons, Inc. (<https://fontawesome.com>), licensed under [CC BY 4.0](https://creativecommons.org/licenses/by/4.0/). The font files are Copyright (c) 2022 Fonticons, Inc., with Reserved Font Name "Font Awesome", under the SIL Open Font License 1.1.
- Material Design Icons by [Pictogrammers](https://pictogrammers.com/), under the Apache License 2.0.
- Powerline Symbols, Copyright 2013 Kim Silkebækken and other contributors, under the MIT licence.
- Hack, Copyright 2018 Source Foundry Authors, under the MIT licence. Bitstream Vera Sans Mono, Copyright 2003 Bitstream, Inc., under the Bitstream Vera License.

The MIT licence in the release zip (`LICENSE`):

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
