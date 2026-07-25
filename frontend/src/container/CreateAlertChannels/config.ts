export interface Channel {
	name: string;
	send_resolved?: boolean;
}

export interface WebhookChannel extends Channel {
	api_url?: string;
	username?: string;
	password?: string;
}

export interface EmailChannel extends Channel {
	to: string;
	html: string;
	headers: Record<string, string>;
}

export type ChannelConfig = Partial<EmailChannel & WebhookChannel>;

export enum ChannelType {
	Email = 'email',
	Webhook = 'webhook',
}
