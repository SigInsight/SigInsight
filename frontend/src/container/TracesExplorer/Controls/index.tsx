import { memo } from 'react';
import { OptionFormatTypes } from 'constants/optionsFormatTypes';
import Controls, { ControlsProps } from 'container/Controls';
import OptionsMenu from 'container/OptionsMenu';
import { OptionsMenuConfig } from 'container/OptionsMenu/types';
import useQueryPagination from 'hooks/queryPagination/useQueryPagination';

import { Container } from './styles';

function TraceExplorerControls({
	isLoading,
	totalCount,
	perPageOptions,
	config,
	showSizeChanger = true,
	hasNextPage,
}: TraceExplorerControlsProps): JSX.Element | null {
	const {
		pagination,
		handleCountItemsPerPageChange,
		handleNavigateNext,
		handleNavigatePrevious,
	} = useQueryPagination(totalCount, perPageOptions, hasNextPage);

	return (
		<Container>
			{config && (
				<OptionsMenu
					selectedOptionFormat={OptionFormatTypes.LIST} // Defaulting it to List view as options are shown only in the List view tab
					config={{ addColumn: config?.addColumn }}
				/>
			)}

			<Controls
				isLoading={isLoading}
				totalCount={totalCount}
				offset={pagination.offset}
				countPerPage={pagination.limit}
				perPageOptions={perPageOptions}
				handleCountItemsPerPageChange={handleCountItemsPerPageChange}
				handleNavigateNext={handleNavigateNext}
				handleNavigatePrevious={handleNavigatePrevious}
				hasNextPage={hasNextPage}
				showSizeChanger={showSizeChanger}
			/>
		</Container>
	);
}

TraceExplorerControls.defaultProps = {
	config: null,
};

type TraceExplorerControlsProps = Pick<
	ControlsProps,
	'isLoading' | 'totalCount' | 'perPageOptions'
> & {
	config?: OptionsMenuConfig | null;
	hasNextPage?: boolean;
	showSizeChanger?: boolean;
};

TraceExplorerControls.defaultProps = {
	showSizeChanger: true,
};

export default memo(TraceExplorerControls);
