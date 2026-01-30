# GTD Templates

## Inbox Item Template
```markdown
---
created: {{date:YYYY-MM-DD}}
status: inbox
type: [task/idea/reference/project]
priority: [low/medium/high/urgent]
tags: []
---

# {{title}}

## Description
Brief description of the item

## Context
Where did this come from? What triggered this thought?

## Next Steps
What needs to happen next?

## Notes
Additional details, links, references
```

## Project Template
```markdown
---
created: {{date:YYYY-MM-DD}}
status: [planning/active/on-hold/completed/cancelled]
type: project
priority: [low/medium/high/urgent]
due_date: 
review_date: 
tags: []
---

# {{title}}

## Purpose & Vision
Why is this project important? What does success look like?

## Outcome
Specific, measurable result that defines "done"

## Next Actions
- [ ] Next concrete physical action
- [ ] Second action (if clear)

## Brainstorming
Ideas, thoughts, possible actions (not committed yet)

## Waiting For
- [ ] Person/Item - Date added

## Resources
Links, documents, references needed

## Notes
Meeting notes, decisions, changes
```

## Next Action Template
```markdown
---
created: {{date:YYYY-MM-DD}}
status: [next/waiting/scheduled/delegated]
context: [@calls/@computer/@errands/@home/@office]
project: "[[Project Name]]"
priority: [low/medium/high/urgent]
due_date: 
tags: []
---

# {{title}}

## Action Required
Clear, specific physical action starting with a verb

## Context
Tools, location, or mindset needed

## Time Estimate
Approximate time needed

## Success Criteria
How will I know this is complete?

## Notes
Additional context or information
```

## Weekly Review Template
```markdown
---
created: {{date:YYYY-MM-DD}}
type: weekly-review
week_of: {{date:YYYY-[W]WW}}
tags: [weekly-review]
---

# Weekly Review - Week of {{date:YYYY-MM-DD}}

## Mind Sweep
What's on my mind that I haven't captured?

## Inbox Processing
- [ ] Process all inbox items to zero
- [ ] Clarify any unclear items

## Calendar Review
### Past Week
What happened? What did I learn?

### Coming Week
What's scheduled? What needs attention?

## Project Review
### Active Projects
Review each project for:
- [ ] Clear next action identified
- [ ] Any stalled projects need attention
- [ ] Any completed projects to archive

### Someday/Maybe Review
- [ ] Any items ready to activate?
- [ ] Any items to remove?

## Next Actions Review
- [ ] Review by context
- [ ] Update stale actions
- [ ] Ensure realistic weekly commitments

## Waiting For Review
- [ ] Follow up on overdue items
- [ ] Update status on pending items

## Goals & Priorities
What are my top 3 priorities for the coming week?
1. 
2. 
3. 

## Reflections
What worked well? What could be improved?
```

## Reference Note Template
```markdown
---
created: {{date:YYYY-MM-DD}}
type: reference
category: [process/contact/document/knowledge]
tags: []
---

# {{title}}

## Summary
Brief overview of what this reference contains

## Content
[Main reference content here]

## Source
Where did this information come from?

## Related
Links to related projects, actions, or references
```

## Waiting For Template
```markdown
---
created: {{date:YYYY-MM-DD}}
status: waiting
waiting_for: "Person/Organization"
date_requested: {{date:YYYY-MM-DD}}
follow_up_date: 
project: "[[Project Name]]"
tags: [waiting-for]
---

# Waiting: {{title}}

## What I'm Waiting For
Specific deliverable or response expected

## Who
Person or organization responsible

## Context
Why am I waiting? What was the request?

## Follow-up Plan
When and how will I follow up if needed?

## Notes
Additional context or communication history
```
