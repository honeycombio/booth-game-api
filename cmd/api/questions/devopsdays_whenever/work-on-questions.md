# Let's brainstorm some questions.

* How do you know what's going on in your software systems? How could that be better?

* Describe your methods of knowing what's happening in your software. How could you have better visibility?

* If your software is failing, how do you know? How do you know what to do about it?

* How does your software tell you what is happening? How could it say that more clearly?

* When software you write or maintain runs in production, how do you know what it struggles with? 

* Your VP messages: production is down. Where do you go?
... it would be more interesting as an ongoing dialog, modeling a real conversation with Martin and me.
Like Eric's interview game, with a point of finding out what they do for o11y.
And delivering that to SalesForce?? ... although they could be lying. 
Or at least we could look through it for clues.

Yeah, like a whole conversation.
After three interactions we give them the option to bail.

OK, yeah, I want that. What can I do today, that is in that direction?

# Objectives for them to reach in the conversation

- tell us how they currently do observability for incidents (100 pts)
    - how do they know when something is wrong?
    - how do they find out what is going on?
    - do they know what actions to take?
- how do they learn a new piece of code?
    - how do you know get familiar with code you didn't write?
    - when you need to change code you haven't worked in before, how do you know you aren't breaking anything?

Then let's take them on the next step of the journey ...
- they understand SLOs
- then understand tracing
- they understand the difference between analysis and search
... I'm not sure how to tell them stuff and get them to express understanding.

## So there would be multiple scorers, like:
- did they tell us what they do for incidents?

# A single question that will let me move in this direction.

for instance.
* Describe your methods of knowing what's happening in your software. How could you have better visibility?

## Scorers

### Score for existing places.
- did they describe how they observe their software? Likely sources include customer complaints; logs; alerts and metrics graphs in dashboards; reading code. Maybe they have distributed tracing. The best answers also include tools that they use for this. Score them from 0 to 20; Give them points for describing how they _currently_ see what is happening in their software."
- Examples:
    - answer: "When an alert happens or a customer complains, we look at dashboards and search logs in Splunk. It could be better if Splunk was faster" 
    response: { "score": 20, "confidence": "high", "reasoning": "they mentioned customers, logs, and a specific tool."  }
    - answer: "blah blah blah" 
    response: { "score": 0, "confidence": "high", "reasoning": "that has nothing to do with observability." }
    - answer: "we test it"
    response: { "score": 5, "confidence": "low", "reasoning": "Their answer might describe how they know their software is working, but not how it is working in production."}

... they could get bonus points for telling us what tools they work in.

... they could get points for saying "customers" "logs" etc; we could have one LLM call pull that out.
... we could get so granular it's "did they say they use logs" ... and only if the text contains "logs"

- also this, but as deterministic code:
... we could simply look for mentions of any of these things and add points, with a simple regex. That would demonstrate some separation of concerns.
metrics, log, dashboards, OpenTelemetry (+5), traces/tracing/trace, APM, Splunk, New Relic, Honeycomb, DataDog, Dynatrace, 
... there are algorithms like Elastic uses, text search algorithms.

// is there a concept of "separate this response into parts" like in image recognition?

### We could classify them into current situations.

Limited observability: they have incomplete logs or metrics
Observability 1.0, Three Pillars: they have alerts on metrics; they can searchable structured logs; perhaps they have APM. They might have 
Observability 2.0, Exploration of Wide Events: they have full distributed tracing using OpenTelemetry, and they can run analysis over those traces; they do structured log analysis; they use event-based Service Level Objectives (SLOs) for alerting; they do dynamic sampling of distributed traces.
Observability 1.5, : they have some characteristics of 2.0 but not all of them. Maybe they do analysis over logs, but don't have traces; or maybe they have traces but they're incomplete or not useful. Maybe they have OpenTelemetry but only use it for metrics.
Other: they did not describe their software observability.

### Score for ideas
- did they have any ideas for making their observability better? Score them between 0 (didn't mention it) and 50 (thoughtful). Give them full points if they want to move to OpenTelemetry or Honeycomb; give them partial credit for wanting tracing or SLOs or better alerts or structured logs.

Examples: 
answer: 

## Response construction

You are Jessitron, an evangelist for great observability. Your goal is to move people and companies from Observability 1.0 (old-style three pillars) to Observability 2.0 (exploratory, with wide events).

You asked them: -question-
They responded with: --

This puts them in the category of --category