import { Form } from 'antd';
import styled from 'styled-components';

export const Label = styled.label`
	font-family: var(--si-font-sans);
	font-size: 13px;
	font-weight: 600;
	line-height: 1;
	letter-spacing: 0;
	color: var(--si-text-primary);
	margin-bottom: 12px;
	display: block;

	.lightMode & {
		color: var(--si-text-primary);
	}
`;

export const FormContainer = styled(Form)`
	display: flex;
	flex-direction: column;
	align-items: flex-start;
	width: 100%;

	& .ant-form-item {
		margin-bottom: 0px;
		width: 100%;

		& .ant-select {
			width: 100%;
			margin: 0;
		}

		& .ant-form-item-control {
			width: 100%;
		}

		& .ant-form-item-control-input {
			width: 100%;
		}

		& .ant-form-item-control-input-content {
			width: 100%;
		}
	}

	& .ant-input,
	& .ant-input-password,
	& .ant-select-selector {
		background: var(--si-bg-subtle) !important;
		border-color: var(--si-border) !important;
		color: var(--si-text-primary) !important;

		.lightMode & {
			background: var(--si-bg-subtle) !important;
			border-color: var(--si-border) !important;
			color: var(--si-text-primary) !important;
		}
	}

	& .ant-input::placeholder {
		color: var(--si-text-muted) !important;

		.lightMode & {
			color: var(--si-text-muted) !important;
		}
	}

	& .ant-input:focus,
	& .ant-input-password:focus,
	& .ant-select-focused .ant-select-selector {
		border-color: var(--si-accent) !important;
		box-shadow: 0 0 0 2px var(--si-accent-soft) !important;
	}
`;

export const ParentContainer = styled.div`
	width: 100%;
	display: flex;
	flex-direction: column;
`;
