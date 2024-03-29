# pack

Simple, multi-threaded texture packer in Go

## Native CLI Usage

### Build

```bash
go build
```

### Run

```bash
./pack -in=images -out=sheets -name=textures
```

For info on more flags, run:

```bash
./pack -h
```

## JS Usage

### Install

#### Yarn

```bash
yarn add --dev @ayebear/pack
```

#### NPM

```bash
npm i -D @ayebear/pack
```

### Run

#### Run with JS wrapper (slower)

```bash
yarn run pack -h
```

#### Run natively (faster)

```bash
./node_modules/@ayebear/pack/pack -h
```

### Client usage

#### Pixi.js v6

##### With static files

Pack outputs a single json file along with (potentially) multiple png files. A pixi.js plugin is included with pack, to load this json file and all associated images in parallel.

```javascript
import { Loader } from '@pixi/loaders'
import { PackSpritesheetLoader } from '@ayebear/pack'

Loader.registerPlugin(PackSpritesheetLoader)
Loader.shared.add('images/textures.json').load(...)
```

##### With bundled files

If you'd like to avoid using static files, and want to "properly" bundle the images and metadata, you can use the `loadSheets` function. The imports for sheetData and sheets might look a bit different depending on your bundler - in this example, parcel v2 is being used:

```javascript
import { Loader } from '@pixi/loaders'
import { loadSheets } from '@ayebear/pack'
import sheetData from 'sheets/textures.json'
import * as sheets from 'sheets/*.png'

// Globbing only gives the "*" part, but we need full path
// to match up with sheetData keys
const images = {}
for (const key in sheets) {
    images[`sheets/${key}.png`] = sheets[key]
}
loadSheets(Loader.shared, sheetData, images).load(...)
```

## About

Pack doesn't do any complicated, slow, puzzle-fitting of sprites. It just bins sprites based on size into separate sprite sheets, in left-to-right grid order. It does this making balanced use of multi-threading to achieve very high performance.

Tested in a real-world project with 300+ sprites. Pack produced 10 sprite sheets in just ~65ms.
