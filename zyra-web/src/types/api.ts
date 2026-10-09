export interface User {
  id: string;
  name: string;
  email: string;
  role: "owner" | "admin" | "trusted" | "normal";
  status: "active" | "retired";
  avatar_url: string;
  appearance: "modern" | "classic";
  theme: "light" | "dark";
}
export interface Session {
  access_token: string;
  refresh_token: string;
  access_expires_at: string;
  session_expires_at: string;
  user: User;
}
export interface Page<T> {
  items: T[];
  total: number;
  page: number;
  limit: number;
}
export interface TicketSummary {
  id: string;
  number: number;
  title: string;
  status: "open" | "closed";
  check_type: string;
  assessment_type: string;
  client_name: string;
  database_name: string;
  created_at: string;
}
export interface Ticket extends TicketSummary {
  client_id: string;
  database_id: string;
  assessment_id?: string;
  hostname: string;
  ip: string;
  evidence: string;
}
export interface TicketDetail {
  ticket: Ticket;
  client: { id: string; name: string; notes?: string };
  database: { id: string; name: string; notes?: string };
  participants: User[];
  saved_ticket: SavedTicket | null;
  similar: Pick<
    TicketSummary,
    "id" | "number" | "title" | "status" | "created_at"
  >[];
}
export interface TicketEvent {
  id: string;
  user_id: string;
  type: string;
  comment?: string;
  created_at: string;
  comment_id?: string;
  content?: CommentContent;
  schema_version?: number;
  revision?: number;
  edited_at?: string;
  mention_users?: MentionOption[];
}

export interface SavedTicket {
  ticket_id: string;
  personal_title: string;
  saved_at: string;
}
export interface SavedTicketSummary extends TicketSummary {
  personal_title: string;
  display_title: string;
  saved_at: string;
}
export interface NotificationSummary {
  id: string;
  ticket_id: string;
  ticket_number: number;
  ticket_title: string;
  comment_id: string;
  actor_id: string;
  actor_name: string;
  actor_avatar_url: string;
  created_at: string;
  read_at: string | null;
}
export interface MentionOption {
  id: string;
  name: string;
  avatar_url: string;
}
export interface CommentInline {
  type: "text" | "mention" | "gif";
  text?: string;
  user_id?: string;
  bold?: boolean;
  italic?: boolean;
  underline?: boolean;
  strike?: boolean;
  size?: 12 | 14 | 16 | 18 | 20 | 24;
  color?: string;
  href?: string;
  src?: string;
  alt?: string;
}
export interface CommentBlock {
  type: "paragraph" | "bullet_list" | "ordered_list" | "code_block";
  align?: "left" | "center" | "right";
  children: CommentInline[];
}
export interface CommentContent { blocks: CommentBlock[]; }
export interface CommentInput {
  schema_version: 1;
  content: CommentContent;
  revision?: number;
}
