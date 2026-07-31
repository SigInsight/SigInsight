import { Button, Tooltip } from 'antd';
import { Plus, Sigma } from 'lucide-react';

import './QueryFooter.styles.scss';

export default function QueryFooter({
	addNewBuilderQuery,
	addNewFormula,
	showAddFormula = true,
}: {
	addNewBuilderQuery: () => void;
	addNewFormula: () => void;
	showAddFormula?: boolean;
}): JSX.Element {
	return (
		<div className="qb-footer">
			<div className="qb-footer-container">
				<div className="qb-add-new-query">
					<Tooltip title={<div style={{ textAlign: 'center' }}>Add New Query</div>}>
						<Button
							className="add-new-query-button periscope-btn "
							icon={<Plus size={16} />}
							onClick={addNewBuilderQuery}
						/>
					</Tooltip>
				</div>

				{showAddFormula && (
					<div className="qb-add-formula">
						<Tooltip
							title={<div style={{ textAlign: 'center' }}>Add New Formula</div>}
						>
							<Button
								className="add-formula-button periscope-btn "
								icon={<Sigma size={16} />}
								onClick={addNewFormula}
							>
								Add Formula
							</Button>
						</Tooltip>
					</div>
				)}
			</div>
		</div>
	);
}
