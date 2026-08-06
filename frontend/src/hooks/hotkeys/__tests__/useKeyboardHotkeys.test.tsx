import { useEffect } from 'react';
import { render } from '@testing-library/react';
import userEvent from '@testing-library/user-event';

import {
	KeyboardHotkeysProvider,
	useKeyboardHotkeys,
} from '../useKeyboardHotkeys';

jest.mock('../../../providers/cmdKProvider', () => ({
	useCmdK: (): { open: boolean } => ({
		open: false,
	}),
}));

function TestComponentWithRegister({
	handleShortcut,
}: {
	handleShortcut: () => void;
}): JSX.Element {
	const { registerShortcut } = useKeyboardHotkeys();

	useEffect(() => {
		registerShortcut('a', handleShortcut);
	}, [registerShortcut, handleShortcut]);

	return <span>Test Component</span>;
}

function TestComponentWithDeRegister({
	handleShortcut,
}: {
	handleShortcut: () => void;
}): JSX.Element {
	const { registerShortcut, deregisterShortcut } = useKeyboardHotkeys();

	useEffect(() => {
		registerShortcut('b', handleShortcut);
		deregisterShortcut('b');
	}, [registerShortcut, deregisterShortcut, handleShortcut]);

	return <span>Test Component</span>;
}

function TestComponentWithCopyShortcut({
	handleShortcut,
}: {
	handleShortcut: () => void;
}): JSX.Element {
	const { registerShortcut } = useKeyboardHotkeys();

	useEffect(() => {
		registerShortcut('meta+c', handleShortcut);
	}, [registerShortcut, handleShortcut]);

	return <span>Test Component</span>;
}

function TestComponentWithClipboardShortcuts({
	handleShortcut,
}: {
	handleShortcut: () => void;
}): JSX.Element {
	const { registerShortcut } = useKeyboardHotkeys();

	useEffect(() => {
		registerShortcut('ctrl+c', handleShortcut);
		registerShortcut('ctrl+v', handleShortcut);
	}, [registerShortcut, handleShortcut]);

	return <input aria-label="clipboard input" />;
}

describe('KeyboardHotkeysProvider', () => {
	it('registers and triggers shortcuts correctly', async () => {
		const handleShortcut = jest.fn();
		const user = userEvent.setup();

		render(
			<KeyboardHotkeysProvider>
				<TestComponentWithRegister handleShortcut={handleShortcut} />
			</KeyboardHotkeysProvider>,
		);

		// fires on keyup
		await user.keyboard('{a}');

		expect(handleShortcut).toHaveBeenCalledTimes(1);
	});

	it('does not trigger deregistered shortcuts', async () => {
		const handleShortcut = jest.fn();
		const user = userEvent.setup();

		render(
			<KeyboardHotkeysProvider>
				<TestComponentWithDeRegister handleShortcut={handleShortcut} />
			</KeyboardHotkeysProvider>,
		);

		await user.keyboard('{b}');

		expect(handleShortcut).not.toHaveBeenCalled();
	});

	it('leaves native copy shortcuts untouched inside a CodeMirror editor', () => {
		const handleShortcut = jest.fn();
		const { container } = render(
			<KeyboardHotkeysProvider>
				<>
					<TestComponentWithCopyShortcut handleShortcut={handleShortcut} />
					<div className="cm-editor">
						<div contentEditable />
					</div>
				</>
			</KeyboardHotkeysProvider>,
		);
		const editor = container.querySelector('.cm-editor') as HTMLElement;
		const copy = new KeyboardEvent('keydown', {
			bubbles: true,
			cancelable: true,
			ctrlKey: true,
			key: 'c',
		});

		expect(editor.dispatchEvent(copy)).toBe(true);
		expect(copy.defaultPrevented).toBe(false);
		expect(handleShortcut).not.toHaveBeenCalled();
	});

	it('leaves native copy and paste shortcuts untouched in regular inputs', () => {
		const handleShortcut = jest.fn();
		const { getByLabelText } = render(
			<KeyboardHotkeysProvider>
				<TestComponentWithClipboardShortcuts handleShortcut={handleShortcut} />
			</KeyboardHotkeysProvider>,
		);
		const input = getByLabelText('clipboard input');

		for (const key of ['c', 'v']) {
			const event = new KeyboardEvent('keydown', {
				bubbles: true,
				cancelable: true,
				ctrlKey: true,
				key,
			});
			expect(input.dispatchEvent(event)).toBe(true);
			expect(event.defaultPrevented).toBe(false);
		}

		expect(handleShortcut).not.toHaveBeenCalled();
	});
});
