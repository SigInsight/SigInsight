// eslint-disable-next-line no-restricted-imports
import { compose, Store } from 'redux';

declare global {
	interface Window {
		store: Store;
		Appcues: Record<string, any>;
		__REDUX_DEVTOOLS_EXTENSION_COMPOSE__: typeof compose;
	}
}

export {};
