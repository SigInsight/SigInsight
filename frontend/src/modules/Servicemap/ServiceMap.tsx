//@ts-nocheck

import { useEffect, useRef } from 'react';
// eslint-disable-next-line no-restricted-imports
import { connect } from 'react-redux';
import { RouteComponentProps, withRouter } from 'react-router-dom';
import { Card } from 'antd';
import Spinner from 'components/Spinner';
import { getDetailedServiceMapItems, ServiceMapStore } from 'store/actions';
import { AppState } from 'store/reducers';
import styled from 'styled-components';
import { GlobalTime } from 'types/actions/globalTime';

import Map from './Map';

const Container = styled.div`
	.force-graph-container {
		overflow: scroll;
	}

	.force-graph-container .graph-tooltip {
		background: black;
		padding: 1px;
		.keyval {
			display: flex;
			.key {
				margin-right: 4px;
			}
			.val {
				margin-left: auto;
			}
		}
	}
`;

interface ServiceMapProps extends RouteComponentProps<any> {
	serviceMap: ServiceMapStore;
	globalTime: GlobalTime;
	getDetailedServiceMapItems: (time: GlobalTime) => void;
}
interface graphNode {
	id: string;
	group: number;
}
interface graphLink {
	source: string;
	target: string;
	value: number;
	callRate: number;
	errorRate: number;
	p99: number;
}
export interface graphDataType {
	nodes: graphNode[];
	links: graphLink[];
}

function ServiceMap(props: ServiceMapProps): JSX.Element {
	const fgRef = useRef();

	const { getDetailedServiceMapItems, globalTime, serviceMap } = props;

	useEffect(() => {
		/*
			Call the apis only when the route is loaded.
			Check this issue: https://github.com/SigInsight/signoz/issues/110
		 */
		getDetailedServiceMapItems(globalTime);
	}, [globalTime, getDetailedServiceMapItems]);

	useEffect(() => {
		fgRef.current && fgRef.current.d3Force('charge').strength(-400);
	});

	if (serviceMap.loading) {
		return <Spinner size="large" tip="Loading..." />;
	}

	if (!serviceMap.loading && serviceMap.items.length === 0) {
		return (
			<Container>
				<Card>No Service Found</Card>
			</Container>
		);
	}
	return (
		<div className="service-map-container">
			<Map fgRef={fgRef} serviceMap={serviceMap} />
		</div>
	);
}

const mapStateToProps = (
	state: AppState,
): {
	serviceMap: ServiceMapStore;
	globalTime: GlobalTime;
} => ({
	serviceMap: state.serviceMap,
	globalTime: state.globalTime,
});

export default withRouter(
	connect(mapStateToProps, {
		getDetailedServiceMapItems,
	})(ServiceMap),
);
