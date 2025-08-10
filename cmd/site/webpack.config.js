const path = require('path');
const glob = require('glob');
const MiniCssExtractPlugin = require('mini-css-extract-plugin');
const { PurgeCSSPlugin } = require('purgecss-webpack-plugin');

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
          'sass-loader'
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
          // /^fa-/, /^svg-inline--fa/, /^sr-only/,
          // /^modal/, /^fade/, /^show/, /^collapse/, /^collapsing/,
          // /^dropdown/, /^tooltip/, /^popover/, /^offcanvas/, /^toast/,
          // /^alert/, /^btn-/, /^bg-/, /^text-/, /^shadow/,
        ]
      }
    })
  ],
  resolve: {
    extensions: ['.js', '.scss', '.css']
  },
  devtool: false,
  stats: 'minimal',
  mode: 'production'
};
