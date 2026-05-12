# Subsystem: Wisdom-Trace (Personalization)

The Trace service tracks the "State of Mind" of each user. It is the bridge between the global Knowledge Graph and the individual learner.

## Use Cases
- **Mastery Tracking**: Update concepts as "Mastered" (@Jesus MASTERED_BY Chess-Opening-A).
- **Fragility Analysis**: Calculate which concepts are likely being forgotten based on time and past review grades.
- **Struggle Identification**: Tag concepts that require review or prerequisite reinforcement.

## User-Specific Logic
Each user has a unique `TraceProfile` stored in the graph using `user_id` namespaces.

### Metrics
- `MasteryScore (0.0-1.0)`: Quantitative measure of success in reviews.
- `RetentionRate`: Based on the Ebbinghaus forgetting curve.
- `Velocity`: How fast the user is progressing through a Curriculum.

## Interface

```protobuf
service Trace {
  rpc RecordEngagement(TraceEvent) returns (TraceUpdate);
  rpc GetWeaknesses(UserRequest) returns (NodeList);
  rpc GetStrengths(UserRequest) returns (NodeList);
}

message TraceEvent {
  string user_id = 1;
  string node_id = 2;
  int32 score = 3; // 1-5 scale
  string context = 4; // "CHAT", "FLASHCARD", "READING"
}
```
