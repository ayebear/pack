import { Spritesheet } from '@pixi/spritesheet'
import { LoaderResource } from '@pixi/loaders'

// Returns true if the object is likely a pack resource
function isPackJson(data) {
	try {
		for (const key in data) {
			if (data[key].sheetSize) {
				return true
			}
		}
	} catch (_) {}
	return false
}

// Convert pack json to pixi json
function packToPixi(data, image) {
	return {
		frames: Object.fromEntries(
			Object.keys(data.sprites).map(key => {
				const pos = data.sprites[key]
				return [
					key,
					{
						sourceSize: { ...data.spriteSize },
						frame: { ...pos, ...data.spriteSize },
						spriteSourceSize: { x: 0, y: 0, ...data.spriteSize },
						rotated: false,
						trimmed: false,
					},
				]
			})
		),
		meta: {
			scale: '1',
			image,
			size: data.sheetSize,
		},
	}
}

// Pixi.js v6 loader
// loader: pixi loader
// data: parsed textures.json, generated from pack
// images: glob imported images object, { image_key: image_filename, ... }
export function loadSheets(loader, data, images) {
	for (const key in data) {
		const meta = data[key]
		const image = images[key]
		if (!image) {
			throw new Error(
				`Image from spritesheet metadata not found for: ${key}`
			)
		}
		loader.add(`${key}_image`, image, resource => {
			const pixiSheet = packToPixi(meta, key)
			const spritesheet = new Spritesheet(
				resource.texture.baseTexture,
				pixiSheet
			)
			spritesheet.parse(() => {
				resource.spritesheet = spritesheet
				resource.textures = spritesheet.textures
			})
		})
	}
	return loader
}

// Pixi.js v6 loader plugin
export class PackSpritesheetLoader {
	static async use(resource, next) {
		if (
			!resource.data ||
			resource.type !== LoaderResource.TYPE.JSON ||
			!isPackJson(resource.data)
		) {
			next()
			return
		}

		const loadOptions = {
			crossOrigin: resource.crossOrigin,
			parentResource: resource,
		}

		// Load each sprite sheet as a resource
		try {
			await Promise.all(
				Object.keys(resource.data).map(async key => {
					return new Promise((resolve, reject) => {
						const metaSheet = resource.data[key]
						const pixiSheet = packToPixi(metaSheet, key)
						this.add(`${key}_image`, key, loadOptions, res => {
							if (res.error) {
								reject(res.error)
								return
							}
							const spritesheet = new Spritesheet(
								res.texture.baseTexture,
								pixiSheet,
								resource.url
							)
							spritesheet.parse(() => {
								resource.spritesheet = spritesheet
								resource.textures = spritesheet.textures
								resolve()
							})
						})
					})
				})
			)
		} catch (error) {
			next(error)
		}
		next()
	}
}
