export const allAlertChannels = [
	{
		id: '3',
		created_at: '2023-08-09T04:45:19.239344617Z',
		updated_at: '2024-06-27T11:37:14.841184399Z',
		name: 'Dummy-Channel',
		type: 'webhook',
		data:
			'{"name":"Dummy-Channel","webhook_configs":[{"url":"https://example.com/alerts","send_resolved":true}]}',
	},
];

export const editAlertChannelInitialValue = {
	url: 'https://example.com/alerts',
	send_resolved: true,
	type: 'webhook',
	name: 'Dummy-Channel',
};
