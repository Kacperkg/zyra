export interface TicketSearch {
  q?: string;
  check?: string;
  page?: number;
  sort?: string;
  order?: string;
  client_id?: string;
  database_id?: string;
  assessment_type?: string;
  number?: string;
  created_from?: string;
  created_to?: string;
}
export const checkLabels: Record<string, string> = {
  backups: "Backups",
  tablespace: "Tablespace",
  filesystem: "Filesystem",
  asm_space: "ASM space",
  archive_destinations: "Archive destinations",
  fra: "Recovery area space",
  missing_email: "Missing email",
  failed_jobs: "Failed jobs",
  datafiles: "Datafiles",
  segments: "Segments",
  extents: "Extents",
  indexes: "Indexes",
  invalid_objects: "Invalid objects",
  clusterware: "Clusterware",
  max_lag: "Max lag",
};
