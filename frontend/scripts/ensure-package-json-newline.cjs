const fs = require('fs');
const path = require('path');

const packageJsonPath = path.resolve(__dirname, '../package.json');
const packageJson = fs.readFileSync(packageJsonPath, 'utf8');

if (!packageJson.endsWith('\n')) {
	fs.appendFileSync(packageJsonPath, '\n');
}
