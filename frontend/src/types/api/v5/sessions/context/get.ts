import { HttpError } from 'types/api';

export interface Props {
	email: string;
	ref: string;
}

export interface PasswordAuthN {
	provider: string;
}

export interface AuthNSupport {
	password: PasswordAuthN[];
}

export interface OrgSessionContext {
	id: string;
	name: string;
	authNSupport: AuthNSupport;
	warning?: HttpError;
}

export interface SessionsContext {
	exists: boolean;
	orgs: OrgSessionContext[];
}
