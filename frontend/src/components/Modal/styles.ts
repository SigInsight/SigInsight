import { Typography } from 'antd';
import styled from 'styled-components';

export const ModalTitle = styled(Typography.Title)`
	font-style: normal;
	font-weight: 600;
	font-size: 1.125rem;
	line-height: 1.5rem;
`;

export const ModalButtonWrapper = styled.div`
	display: flex;
	flex-direction: row-reverse;
	gap: 0.625rem;
`;
