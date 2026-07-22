import { Modal, ModalProps } from 'antd';

import './AppModal.style.scss';

function AppModal({
	children,
	width = 672,
	rootClassName = '',
	...rest
}: ModalProps): JSX.Element {
	return (
		<Modal
			centered
			width={width}
			cancelText="Close"
			rootClassName={`app-modal ${rootClassName}`}
			{...rest}
		>
			{children}
		</Modal>
	);
}

export default AppModal;
