const path = require("path");

class FaviconsPartialPlugin {
  constructor(opts = {}) {
    this.opts = {
      logo: "",
      publicPath: "/static/assets",
      outputPath: "assets",
      partialFilename: "favicons.partial.html",
      faviconsOptions: {
        appName: "",
        appDescription: "",
        developerName: "",
        background: "#ddd",
        theme_color: "#000",
        icons: { coast: false, yandex: false },
      },
      ...opts,
    };
  }

  apply(compiler) {
    const { Compilation, sources } = compiler.webpack;

    compiler.hooks.thisCompilation.tap("FaviconsPartialPlugin", (compilation) => {
      compilation.hooks.processAssets.tapPromise(
        { name: "FaviconsPartialPlugin", stage: Compilation.PROCESS_ASSETS_STAGE_ADDITIONS },
        async () => {
          const mod = await import("favicons");
          const favicons = mod.favicons ?? mod.default ?? mod;

          const logoAbs = path.resolve(compiler.context, this.opts.logo);
          compilation.fileDependencies.add(logoAbs);

          const res = await favicons(logoAbs, {
            path: this.opts.publicPath,
            ...this.opts.faviconsOptions,
          });

          for (const img of res.images) {
            compilation.emitAsset(
              path.posix.join(this.opts.outputPath, img.name),
              new sources.RawSource(img.contents)
            );
          }
          for (const file of res.files) {
            compilation.emitAsset(
              path.posix.join(this.opts.outputPath, file.name),
              new sources.RawSource(file.contents)
            );
          }

          compilation.emitAsset(
            this.opts.partialFilename,
            new sources.RawSource(res.html.join("\n"))
          );
        }
      );
    });
  }
}

module.exports = FaviconsPartialPlugin;
