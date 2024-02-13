const path = require('path');

module.exports = function (_env, argv) {
	const isProduction = argv.mode === 'production';

	return {
		devtool: !isProduction && 'cheap-module-source-map',
		entry: './src/index.jsx',
		output: {
			path: path.resolve(__dirname, 'public'),
			filename: '[name].js',
			publicPath: '/',
		},
		stats: 'minimal',
		module: {
			rules: [
				{
					test: /\.jsx?$/,
					exclude: /node_modules/,
					use: {
						loader: 'babel-loader',
						options: {
							cacheDirectory: true,
							cacheCompression: false,
							envName: isProduction ? 'production' : 'development',
						},
					},
				},
				{
					test: /\.css$/,
					use: ['style-loader', 'css-loader'],
				},
			],
		},
		resolve: {
			extensions: ['.js', '.jsx'],
		},
		devServer: {
			setupExitSignals: true,
		},
	};
};
