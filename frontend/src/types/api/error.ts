import { StatusCodes } from 'http-status-codes';

import { HttpErrorResponse } from '.';

class APIError extends Error {
	error: HttpErrorResponse;

	constructor(error: HttpErrorResponse) {
		super(error.error.message);
		this.error = error;
	}

	getHttpStatusCode(): StatusCodes {
		return this.error.httpStatusCode;
	}

	getErrorMessage(): string {
		return this.error.error.message;
	}

	getErrorCode(): string {
		return this.error.error.code;
	}

	getErrorDetails(): HttpErrorResponse {
		return this.error;
	}
}

export default APIError;
