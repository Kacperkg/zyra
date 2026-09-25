export interface User {
  id: string;
  name: string;
  email: string;
  role: "admin" | "trusted" | "normal";
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
}
