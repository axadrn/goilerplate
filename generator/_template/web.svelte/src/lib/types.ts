// JSON shapes of the API under /api/v1. openapi.yaml in internal/api is the
// reference; these types cover what the app reads.

export interface User {
	id: string;
	email: string;
	name: string;
	password_set: boolean;
	created_at: string;
}

export interface Account {
	user: User;
	onboarding_required: boolean;
	// Present with storage.
	avatar_url?: string | null;
	// Present with workspaces.
	workspaces?: Membership[];
	// Present in Paid builds without workspaces.
	subscription?: Subscription | null;
	access?: Access | null;
	// Present in Paid builds: whether this session belongs to an operator.
	is_admin?: boolean;
	// Present with workspaces: an invitation this browser opened before
	// signing in, from the pending_invitation cookie.
	pending_invitation?: string | null;
}

export interface Meta {
	name: string;
	support_email: string;
}

export type WorkspaceRole = 'owner' | 'admin' | 'member';

export interface Workspace {
	id: string;
	name: string;
	slug: string;
}

export interface Membership {
	workspace: Workspace;
	role: WorkspaceRole;
}

export interface WorkspacePermissions {
	manage_workspace: boolean;
	manage_members: boolean;
	manage_owners: boolean;
	manage_billing: boolean;
	delete_workspace: boolean;
	view_audit_log: boolean;
}

export interface WorkspaceMember {
	id: string;
	user_id: string;
	name: string;
	// Empty for viewers who may not manage members.
	email?: string;
	role: WorkspaceRole;
	created_at: string;
}

export interface PendingInvitation {
	id: string;
	email: string;
	role: WorkspaceRole;
	expires_at: string;
	created_at: string;
}

export interface Seats {
	used: number;
	// Null when the plan has no member limit.
	limit: number | null;
}

export interface WorkspaceDetail {
	workspace: Workspace;
	role: WorkspaceRole;
	permissions: WorkspacePermissions;
	members: WorkspaceMember[];
	invitations: PendingInvitation[];
	seats: Seats;
}

export interface WorkspaceEvent {
	id: string;
	actor_user_id: string | null;
	actor_email: string;
	action: string;
	target_type: string;
	target_id: string;
	metadata: string;
	created_at: string;
}

export interface ActivityPage {
	items: WorkspaceEvent[];
	// Continues after the last event; null on the last page.
	next_cursor: string | null;
}

export interface InvitationPreview {
	workspace_name: string;
	inviter_name: string;
	role: WorkspaceRole;
	expires_at: string;
}

export interface CSRFToken {
	token: string;
}

// The answer of a cookie sign-in (?session=cookie): the session lives in the
// HttpOnly session_token cookie, never in the body.
export interface CookieSession {
	expires_at: string;
	user: User;
}

export interface AccountSession {
	id: string;
	display_name: string;
	current: boolean;
	last_seen_at: string;
	created_at: string;
}

export interface List<T> {
	items: T[];
}

export type TokenScope = 'read' | 'write';

export interface APIToken {
	id: string;
	name: string;
	prefix: string;
	scopes: TokenScope[];
	// Present with workspaces.
	workspace_id?: string;
	last_used_at: string | null;
	expires_at: string | null;
	expired: boolean;
	created_at: string;
}

export interface CreatedAPIToken {
	token: string;
	api_token: APIToken;
}

export interface AvatarAnswer {
	avatar_url: string | null;
}

export type SubscriptionStatus =
	| 'active'
	| 'trialing'
	| 'past_due'
	| 'incomplete'
	| 'inactive'
	| 'cancelled'
	| 'blocked';

export interface Subscription {
	id: string;
	plan_id: string;
	status: SubscriptionStatus;
	// Empty until the payment provider knows a customer for it.
	provider: string;
	current_period_end: string | null;
	amount: number | null;
	currency: string;
	interval: BillingInterval | null;
	created_at: string;
	updated_at: string;
}

export type AccessState = 'free' | 'trial' | 'paid' | 'grace' | 'blocked';

export interface Access {
	plan_id: string;
	state: AccessState;
	trial_ends_at: string | null;
	access_ends_at: string | null;
}

export type BillingInterval = 'monthly' | 'yearly';

export interface PlanPrice {
	interval: BillingInterval;
	amount: number;
	display_price: string;
}

export interface Plan {
	id: string;
	name: string;
	description: string;
	benefits: string[];
	current: boolean;
	prices: PlanPrice[];
}

export interface Billing {
	subscription: Subscription | null;
	access: Access | null;
	plans: Plan[];
}

export interface URLAnswer {
	url: string;
}

export interface AdminMetrics {
	users: number;
	verified_users: number;
	// Present with workspaces.
	workspaces?: number;
	paid_workspaces?: number;
	// Present without workspaces.
	paid_users?: number;
	active_sessions: number;
}

export interface AdminUser {
	id: string;
	email: string;
	name: string;
	email_verified_at: string | null;
	created_at: string;
	// Present with workspaces.
	workspace_count?: number;
	active_session_count: number;
	is_admin: boolean;
}

export interface AdminUserPage {
	items: AdminUser[];
	query: string;
	page: number;
	total_pages: number;
	total_users: number;
}

export interface AdminAuditEvent {
	id: string;
	actor_user_id: string | null;
	actor_email: string;
	action: string;
	target_type: string;
	target_id: string;
	metadata: string;
	created_at: string;
}

export interface AdminDashboard {
	metrics: AdminMetrics;
	users: AdminUserPage;
	events: AdminAuditEvent[];
}
