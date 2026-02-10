const path = require('path');
const glob = require('glob');
const MiniCssExtractPlugin = require('mini-css-extract-plugin');
const { PurgeCSSPlugin } = require('purgecss-webpack-plugin');
const FaviconsPartialPlugin = require("./build/FaviconsPartialPlugin");

const PATHS = {
  pages: path.join(__dirname, 'pages'),
  frontend: path.join(__dirname, 'frontend')
};

module.exports = {
  entry: path.join(PATHS.frontend, 'index.js'),
  output: {
    path: path.resolve(__dirname, 'static'),
    filename: 'bundle.js',
    clean: true
  },
  module: {
    rules: [
      {
        test: /\.scss$/,
        use: [
          MiniCssExtractPlugin.loader,
          {
            loader: 'css-loader',
            options: { url: true, importLoaders: 1 }
          },
          {
            loader: 'sass-loader',
            options: {
              api: 'modern-compiler',
              sassOptions: {
                silenceDeprecations: ['import', 'global-builtin', 'color-functions', 'mixed-decls'],
                quietDeps: true,
              }
            }
          }
        ]
      },
      {
        test: /\.css$/,
        use: [MiniCssExtractPlugin.loader, 'css-loader']
      },
      {
        test: /\.(woff2?|ttf|eot|otf)$/,
        type: 'asset/resource',
        generator: { filename: 'fonts/[name][ext]' }
      },
      {
        test: /\.(png|jpe?g|gif|svg)$/,
        type: 'asset/resource',
        generator: { filename: 'images/[name][ext]' }
      }
    ]
  },
  plugins: [
    new MiniCssExtractPlugin({ filename: 'style.css' }),
    new PurgeCSSPlugin({
      paths: [
        ...glob.sync(`${PATHS.pages}/**/*`, { nodir: true }),
        ...glob.sync(`${PATHS.frontend}/**/*`, { nodir: true })
      ],
      safelist: {
        standard: [
          "tbody", "tfoot",
        ]
      }
    }),
    new FaviconsPartialPlugin({
      logo: './icons/siren.svg',
      partialFilename: "../partial/favicons.partial.html",
      favicons: {
        appName: 'SIREN',
        appDescription: 'The Telegram bot for webcast alerts',
        developerName: 'bcmk',
        background: 'transparent',
        theme_color: '#ffffff',
        start_url: '/',
        icons: {
          coast: false,
          yandex: false,
        }
      }
    }),
  ],
  resolve: {
    extensions: ['.js', '.scss', '.css']
  },
  performance: {
    maxAssetSize: 512000,
    maxEntrypointSize: 512000,
  },
  devtool: false,
  stats: 'minimal',
  mode: 'production'
};
