import { useEffect, useState } from 'react';
import { Button } from '@signozhq/button';
import { Checkbox } from '@signozhq/checkbox';
import { Input } from '@signozhq/input';
import { Input as AntdInput } from 'antd';
import logEvent from 'api/common/logEvent';
import { ArrowRight } from 'lucide-react';

import { OnboardingQuestionHeader } from '../OnboardingQuestionHeader';

import '../OnboardingQuestionaire.styles.scss';

export interface ProductDetails {
	interestInProduct: string[] | null;
	otherInterestInProduct: string | null;
	discoverProduct: string | null;
}

interface AboutProductQuestionsProps {
	productDetails: ProductDetails;
	setProductDetails: (details: ProductDetails) => void;
	onNext: () => void;
}

const interestedInOptions: Record<string, string> = {
	loweringCosts: 'Lowering observability costs',
	otelNativeStack: 'Interested in OTel-native stack',
	deploymentFlexibility: 'Deployment flexibility (Cloud/Self-Host) in future',
	singleTool:
		'Single Tool (logs, metrics & traces) to reduce operational overhead',
	correlateSignals: 'Correlate signals for faster troubleshooting',
};

export function AboutProductQuestions({
	productDetails,
	setProductDetails,
	onNext,
}: AboutProductQuestionsProps): JSX.Element {
	const [interestInProduct, setInterestInProduct] = useState<string[]>(
		productDetails?.interestInProduct || [],
	);
	const [otherInterestInProduct, setOtherInterestInProduct] = useState<string>(
		productDetails?.otherInterestInProduct || '',
	);
	const [discoverProduct, setDiscoverProduct] = useState<string>(
		productDetails?.discoverProduct || '',
	);
	const [isNextDisabled, setIsNextDisabled] = useState<boolean>(true);

	useEffect((): void => {
		if (
			discoverProduct !== '' &&
			interestInProduct.length > 0 &&
			(!interestInProduct.includes('Others') || otherInterestInProduct !== '')
		) {
			setIsNextDisabled(false);
		} else {
			setIsNextDisabled(true);
		}
	}, [interestInProduct, otherInterestInProduct, discoverProduct]);

	const handleInterestChange = (option: string, checked: boolean): void => {
		if (checked) {
			setInterestInProduct((prev) => [...prev, option]);
		} else {
			setInterestInProduct((prev) => prev.filter((item) => item !== option));
		}
	};

	const createInterestChangeHandler = (option: string) => (
		checked: boolean,
	): void => {
		handleInterestChange(option, Boolean(checked));
	};

	const handleOnNext = (): void => {
		setProductDetails({
			discoverProduct,
			interestInProduct,
			otherInterestInProduct,
		});

		logEvent('Org Onboarding: Answered', {
			discoverProduct,
			interestInProduct,
			otherInterestInProduct,
		});

		onNext();
	};

	return (
		<div className="questions-container">
			<OnboardingQuestionHeader
				title="Set up your workspace"
				subtitle="Tailor SigInsight to suit your observability needs."
			/>

			<div className="questions-form-container">
				<div className="questions-form">
					<div className="form-group">
						<div className="question">How did you first come across SigInsight?</div>

						<AntdInput.TextArea
							className="discover-product-input"
							placeholder={`e.g., googling "datadog alternative", a post on r/devops, from a friend/colleague, a LinkedIn post, ChatGPT, etc.`}
							value={discoverProduct}
							autoFocus
							rows={4}
							onChange={(e): void => setDiscoverProduct(e.target.value)}
						/>
					</div>

					<div className="form-group">
						<div className="question">What got you interested in SigInsight?</div>
						<div className="checkbox-grid">
							{Object.keys(interestedInOptions).map((option: string) => (
								<div key={option} className="checkbox-item">
									<Checkbox
										id={`checkbox-${option}`}
										checked={interestInProduct.includes(option)}
										onCheckedChange={createInterestChangeHandler(option)}
										labelName={interestedInOptions[option]}
									/>
								</div>
							))}

							<div className="checkbox-item checkbox-item-others">
								<Checkbox
									id="others-checkbox"
									checked={interestInProduct.includes('Others')}
									onCheckedChange={createInterestChangeHandler('Others')}
									labelName={interestInProduct.includes('Others') ? '' : 'Others'}
								/>
								{interestInProduct.includes('Others') && (
									<Input
										type="text"
										className="onboarding-questionaire-other-input"
										placeholder="What got you interested in SigInsight?"
										value={otherInterestInProduct}
										autoFocus
										onChange={(e): void => setOtherInterestInProduct(e.target.value)}
									/>
								)}
							</div>
						</div>
					</div>
				</div>

				<div className="onboarding-buttons-container">
					<Button
						variant="solid"
						color="primary"
						className={`onboarding-next-button ${isNextDisabled ? 'disabled' : ''}`}
						onClick={handleOnNext}
						disabled={isNextDisabled}
						suffixIcon={<ArrowRight size={12} />}
					>
						Next
					</Button>
				</div>
			</div>
		</div>
	);
}
