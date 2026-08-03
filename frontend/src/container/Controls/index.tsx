import { memo, useMemo } from 'react';
import { LeftOutlined, RightOutlined } from '@ant-design/icons';
import { Button, Select } from 'antd';
import { DEFAULT_PER_PAGE_OPTIONS, Pagination } from 'hooks/queryPagination';
import { popupContainer } from 'utils/selectPopupContainer';

import { defaultSelectStyle } from './config';
import { Container } from './styles';

function Controls({
	offset = 0,
	perPageOptions = DEFAULT_PER_PAGE_OPTIONS,
	isLoading,
	totalCount,
	countPerPage,
	handleNavigatePrevious,
	handleNavigateNext,
	handleCountItemsPerPageChange,
	isLogPanel = false,
	showSizeChanger = true,
	hasNextPage,
}: ControlsProps): JSX.Element | null {
	const isNavigationDisabled = useMemo(() => isLoading || countPerPage < 0, [
		isLoading,
		countPerPage,
	]);
	const isPreviousDisabled = useMemo(
		() => (isLogPanel ? false : offset <= 0 || isNavigationDisabled),
		[isLogPanel, isNavigationDisabled, offset],
	);
	const isNextDisabled = useMemo(
		() =>
			isLogPanel
				? false
				: isNavigationDisabled ||
				  (hasNextPage === undefined
						? totalCount === 0 || totalCount < countPerPage
						: !hasNextPage),
		[countPerPage, hasNextPage, isLogPanel, isNavigationDisabled, totalCount],
	);

	return (
		<Container>
			<Button
				loading={isLoading}
				size="small"
				type="link"
				disabled={isPreviousDisabled}
				onClick={handleNavigatePrevious}
			>
				<LeftOutlined /> Previous
			</Button>
			<Button
				loading={isLoading}
				size="small"
				type="link"
				disabled={isNextDisabled}
				onClick={handleNavigateNext}
			>
				Next <RightOutlined />
			</Button>

			{showSizeChanger && (
				<Select<Pagination['limit']>
					style={defaultSelectStyle}
					loading={isLoading}
					value={countPerPage}
					onChange={handleCountItemsPerPageChange}
					getPopupContainer={popupContainer}
				>
					{perPageOptions.map((count) => (
						<Select.Option
							key={count}
							value={count}
						>{`${count} / page`}</Select.Option>
					))}
				</Select>
			)}
		</Container>
	);
}

Controls.defaultProps = {
	offset: 0,
	perPageOptions: DEFAULT_PER_PAGE_OPTIONS,
	isLogPanel: false,
	showSizeChanger: true,
};

export interface ControlsProps {
	offset?: Pagination['offset'];
	perPageOptions?: number[];
	totalCount: number;
	countPerPage: Pagination['limit'];
	isLoading: boolean;
	handleNavigatePrevious: () => void;
	handleNavigateNext: () => void;
	handleCountItemsPerPageChange: (value: Pagination['limit']) => void;
	isLogPanel?: boolean;
	showSizeChanger?: boolean;
	hasNextPage?: boolean;
}

export default memo(Controls);
