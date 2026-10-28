package harness

default allow := false

allow if {
  input.action == "task.read"
  input.subject.tenant_id == input.resource.tenant_id
}

allow if {
  input.action == "assignment.create"
  input.subject.tenant_id == input.resource.tenant_id
  input.subject.kind == "AGENT"
}
