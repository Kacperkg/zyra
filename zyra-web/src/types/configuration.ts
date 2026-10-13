import type { TicketSummary } from "./api";
export interface ClientRecord {
  id: string;
  name: string;
  notes: string;
  archived: boolean;
}
export interface DatabaseRecord {
  id: string;
  client_id: string;
  name: string;
  hostname: string;
  ip: string;
  notes: string;
  archived: boolean;
  engine: "oracle" | "sql";
  max_lag_sequences: number | null;
  observed_unique_id: string;
  observed_unique_id_assessment_id: string;
  observed_unique_id_received_at: string | null;
}
export interface ResourceRule {
  ignore: boolean;
  min_free_percent?: number;
  max_used_percent?: number;
}
export interface CheckSettings {
  selected: Record<string, boolean>;
  resources: Record<string, Record<string, ResourceRule>>;
}
export interface SourceSchedule {
  name: string;
  start: string;
  end: string;
}
export interface EmailSource {
  id: string;
  database_id: string;
  subject: string;
  sender: string;
  mailbox: string;
  assessment_type: "daily_check" | "standby_check";
  timezone: string;
  enabled: boolean;
  schedules: SourceSchedule[];
  weekdays: string[];
  effective_at: string;
}
export interface AssessmentSummary {
  id: string;
  database_id: string;
  type: string;
  received_at: string;
  report_time: string;
  hostname: string;
  ip: string;
  unique_id: string;
  result: string;
  ticket_count: number;
  tickets: Pick<
    TicketSummary,
    "id" | "number" | "title" | "status" | "check_type" | "created_at"
  >[];
}
