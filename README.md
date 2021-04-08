# pack

Simple, multi-threaded texture packer in Go

## Native CLI Usage

### Build

```bash
go build
```

### Run

```bash
./pack -in=images -out=images_out -name=textures
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

### Client usage (Pixi.js v6)

Pack outputs a single json file along with (potentially) multiple png files. A pixi.js plugin is included with pack, to load this json file and all associated images in parallel.

```javascript
import { Loader } from '@pixi/loaders'
import { PackSpritesheetLoader } from '@ayebear/pack'

Loader.registerPlugin(PackSpritesheetLoader)
Loader.shared.add('images/textures.json').load(...)
```

## About

Pack doesn't do any complicated, slow, puzzle-fitting of sprites. It just bins sprites based on size into separate sprite sheets, in left-to-right grid order. It does this making balanced use of multi-threading to achieve very high performance.

Tested in a real-world project with 300+ sprites. Pack produced 10 sprite sheets in just ~65ms.
