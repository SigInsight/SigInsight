import { UPlotChartProps } from './components/types';
import UPlotChart from './components/UPlotChart/UPlotChart';
import { PlotContextProvider } from './context/PlotContext';

export function UPlotChartHost(props: UPlotChartProps): JSX.Element {
	return (
		<PlotContextProvider>
			<UPlotChart {...props} />
		</PlotContextProvider>
	);
}
