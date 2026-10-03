# 4sight design

## Agent Prompts and Agent Responses

By convention, some commands/subcommands may have a nested "agent prompt" and
"agent response" subcommands.

The "agent prompt"" command output provides an agent prompt, which the agent is
to read for information and instructions.

If the agent is to provide information back, it can call various commands, but
the "agent reponse" for the same parent command has some coupling with "agent
prompt"

### Coupling of prompt and response

The "agent reponse" may need more data than comfortably fits in additional
command line arguments. In this case, during an implementation, ensure the
"agent response" implementation has an encoding scheme for everything it needs,
and that the "agent prompt" instructs the agent on the details of the
specification.

The "agent response" command if writing to the db, must receive enough data and
inferable (or concrete) references to satisfy db spec constraints, some of which
are not constrained in schema and need to be constrained by the "agent response"

The "agent prompt" implementation must always instruct agent how to respond
properly.

The actual prompt/response spec is an internal implementation detail and not
part of application specification. If it needs a schema itself (such as a json
schema), then write it to `.4sight/<command path>/agent/responseSchema.<filetype
for schema definition>` so that the implementation of the response can interpret
it and the prompt can provide it to agent. The schema should include as many
constraints specified by the db schema, if can't natively be done in the
response schema then provide as value descriptions.

### Agent Prompt

Some additional rules when interpreting the agent promp spec:

When we say "the agent should XXX" or "instruct agent to XXX" it is meant the
output prompt should be instructing the agent to do XXX".

When we say the user should XX or instruct user to XX, it means the output
prompt should be instructing the agent to instruct the user to XX.

When we say to "stop" it is meant the output prompt should wrap up and we stop
processing, generally an error condition unnless at the end.
