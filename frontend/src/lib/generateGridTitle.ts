import React, { ReactNode } from 'react';

function nodeText(node: ReactNode): string {
	if (typeof node === 'string' || typeof node === 'number') {
		return String(node);
	}
	if (Array.isArray(node)) {
		return node.map(nodeText).filter(Boolean).join(' ');
	}
	if (React.isValidElement<{ children?: ReactNode }>(node)) {
		return nodeText(node.props.children);
	}
	return '';
}

export const generateGridTitle = (title: ReactNode): string =>
	nodeText(title).replace(/\s+/g, ' ').trim();
