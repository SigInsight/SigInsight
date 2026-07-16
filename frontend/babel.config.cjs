module.exports = {
	presets: [
		['@babel/preset-env', { modules: 'auto' }],
		['@babel/preset-react', { runtime: 'automatic' }],
	],
	env: {
		test: {
			presets: [
				[
					'@babel/preset-env',
					{ modules: 'commonjs', targets: { node: 'current' } },
				],
				['@babel/preset-react', { runtime: 'automatic' }],
			],
		},
	},
};
