import { Space, Typography } from 'antd';
import UnAuthorized from 'assets/UnAuthorized';
import { Container } from 'components/NotFound/styles';
import { useQueryState } from 'nuqs';

import { useAppContext } from '../../providers/App/App';
import { USER_ROLES } from '../../types/roles';

import './index.styles.scss';

function UnAuthorizePage(): JSX.Element {
	const [debugCurrentRole] = useQueryState('currentRole');
	const { user } = useAppContext();

	const userIsAnonymous =
		debugCurrentRole === USER_ROLES.ANONYMOUS ||
		user.role === USER_ROLES.ANONYMOUS;
	const mistakeMessage = 'Please contact your administrator.';

	return (
		<Container className="unauthorized-page">
			<Space align="center" direction="vertical">
				<UnAuthorized width={64} height={64} />
				<Typography.Title level={3}>Access Restricted</Typography.Title>

				<p className="unauthorized-page__description">
					It looks like you don&lsquo;t have permission to view this page. <br />
					{mistakeMessage}
				</p>
			</Space>
		</Container>
	);
}

export default UnAuthorizePage;
