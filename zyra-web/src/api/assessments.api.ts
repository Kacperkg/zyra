import { request } from "./client";
export interface AssessmentMetadata {
  id: string;
  subject: string;
  sender: string;
  received_at: string;
  report_time: string;
}
export const assessmentsApi = {
  detail: (id: string) =>
    request<AssessmentMetadata>(`/assessments/${encodeURIComponent(id)}`),
};
