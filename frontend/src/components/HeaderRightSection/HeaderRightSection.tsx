import { useCallback, useState } from 'react';
import { useLocation } from 'react-router-dom';
import { Button, Popover } from 'antd';
import logEvent from 'api/common/logEvent';
import { Globe, Inbox } from 'lucide-react';

import AnnouncementsModal from './AnnouncementsModal';
import ShareURLModal from './ShareURLModal';

import './HeaderRightSection.styles.scss';

interface HeaderRightSectionProps {
	enableAnnouncements: boolean;
	enableShare: boolean;
}

function HeaderRightSection({
	enableAnnouncements,
	enableShare,
}: HeaderRightSectionProps): JSX.Element | null {
	const location = useLocation();

	const [openShareURLModal, setOpenShareURLModal] = useState(false);
	const [openAnnouncementsModal, setOpenAnnouncementsModal] = useState(false);

	const handleOpenShareURLModal = useCallback((): void => {
		logEvent('Share: Clicked', {
			page: location.pathname,
		});

		setOpenShareURLModal(true);
		setOpenAnnouncementsModal(false);
	}, [location.pathname]);

	const handleOpenAnnouncementsModalChange = (open: boolean): void => {
		setOpenAnnouncementsModal(open);
	};

	const handleOpenShareURLModalChange = (open: boolean): void => {
		setOpenShareURLModal(open);
	};

	return (
		<div className="header-right-section-container">
			{enableAnnouncements && (
				<Popover
					rootClassName="header-section-popover-root"
					className="shareable-link-popover"
					placement="bottomRight"
					content={<AnnouncementsModal />}
					arrow={false}
					destroyTooltipOnHide
					trigger="click"
					open={openAnnouncementsModal}
					onOpenChange={handleOpenAnnouncementsModalChange}
				>
					<Button
						icon={<Inbox size={14} />}
						className="periscope-btn ghost announcements-btn"
						onClick={(): void => {
							logEvent('Announcements: Clicked', {
								page: location.pathname,
							});
						}}
					/>
				</Popover>
			)}

			{enableShare && (
				<Popover
					rootClassName="header-section-popover-root"
					className="shareable-link-popover"
					placement="bottomRight"
					content={<ShareURLModal />}
					open={openShareURLModal}
					destroyTooltipOnHide
					arrow={false}
					trigger="click"
					onOpenChange={handleOpenShareURLModalChange}
				>
					<Button
						className="share-link-btn periscope-btn ghost"
						icon={<Globe size={14} />}
						onClick={handleOpenShareURLModal}
					>
						Share
					</Button>
				</Popover>
			)}
		</div>
	);
}

export default HeaderRightSection;
