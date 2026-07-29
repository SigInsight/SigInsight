import { Button, Typography } from 'antd';
import Modal from 'components/Modal';

function SkipOnBoardingModal({ onContinueClick }: Props): JSX.Element {
	return (
		<Modal
			title="Setup instrumentation"
			isModalVisible
			closable={false}
			footer={[
				<Button key="submit" type="primary" onClick={onContinueClick}>
					Continue without instrumentation
				</Button>,
			]}
		>
			<div>
				<Typography>No instrumentation data.</Typography>
				<Typography>Please instrument your application to continue.</Typography>
			</div>
		</Modal>
	);
}

interface Props {
	onContinueClick: () => void;
}

export default SkipOnBoardingModal;
