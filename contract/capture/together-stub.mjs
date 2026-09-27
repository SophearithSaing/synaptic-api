/**
 * Deterministic Together API stub (HTTP-level interception).
 *
 * Replaces `globalThis.fetch` before the Nest app boots so the
 * `together-ai` SDK (which captures the global fetch at client
 * construction) never performs real network calls. Requests to
 * `api.together.com` are answered with fixed, schema-valid payloads;
 * everything else is delegated to the real fetch.
 *
 * The active stub mode lives in `stubState.mode` and is switched by the
 * harness between scenarios (process-level control, no app changes):
 *
 * - `ok`          valid completion; generation requests return a question
 *                 matching the requested questionType, evaluation
 *                 requests return one evaluation per submitted answer
 *                 (score 0 when the answer starts with "wrong", else 1)
 * - `invalid-json` completion content is not JSON -> app 503
 * - `empty`        completion content is null -> app 503
 * - `http-error`   HTTP 500 from the provider -> app 500
 */
const TOGETHER_HOSTS = /(^|\.)together\.ai$/;

export const stubState = {
  mode: 'ok',
  /** Number of intercepted Together calls, for harness diagnostics. */
  calls: 0,
};

/** Builds a fixed generated question for the requested type. */
function buildGeneratedQuestion(requestBody) {
  const userPrompt = JSON.parse(requestBody.messages[1].content);
  const questionType =
    userPrompt.questionType === 'written' ? 'written' : 'mcq';
  const question = {
    id: 'stub-q',
    type: questionType,
    prompt: `Stub ${questionType} question (${userPrompt.topicSlug} level ${userPrompt.level})`,
    targetConcepts: ['stub-concept'],
    feedback: {
      correct: 'Stub correct feedback.',
      incorrect: 'Stub incorrect feedback.',
    },
    rubrics: {
      keyPoints: ['Stub key point.'],
      misconceptions: ['Stub misconception.'],
    },
  };

  if (questionType === 'mcq') {
    question.options = [
      { id: 'o1', text: 'Stub option one' },
      { id: 'o2', text: 'Stub option two' },
      { id: 'o3', text: 'Stub option three' },
    ];
    question.correctOptionId = 'o2';
  }

  return { question };
}

/** Builds one deterministic evaluation per submitted written answer. */
function buildEvaluations(requestBody) {
  const userPrompt = JSON.parse(requestBody.messages[1].content);

  return {
    evaluations: userPrompt.answers.map((answer) => {
      const failed = answer.studentAnswer.trimStart().startsWith('wrong');
      return {
        questionId: answer.questionId,
        score: failed ? 0 : 1,
        correctAnswer: 'Stub correct answer.',
        feedback: failed ? 'Stub failing feedback.' : 'Stub passing feedback.',
        strengths: failed ? [] : ['stub-concept'],
        weaknesses: failed ? ['stub-concept'] : [],
      };
    }),
  };
}

/** Builds a chat.completion response body for the current mode. */
function buildCompletion(requestBody) {
  const schemaName = requestBody.response_format?.json_schema?.name;
  const payload =
    schemaName === 'written_answer_evaluations'
      ? buildEvaluations(requestBody)
      : buildGeneratedQuestion(requestBody);

  return {
    id: 'chatcmpl-stub',
    object: 'chat.completion',
    created: 0,
    model: requestBody.model ?? 'stub-model',
    choices: [
      {
        index: 0,
        message: { role: 'assistant', content: JSON.stringify(payload) },
        finish_reason: 'stop',
      },
    ],
    usage: {
      prompt_tokens: 1,
      completion_tokens: 1,
      total_tokens: 2,
    },
  };
}

function jsonResponse(status, body) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { 'content-type': 'application/json' },
  });
}

/**
 * Installs the stub over `globalThis.fetch`. Returns the original fetch
 * so the harness HTTP client can keep using real network access.
 */
export function installTogetherStub() {
  const realFetch = globalThis.fetch;

  globalThis.fetch = async (url, options = {}) => {
    const host = new URL(url).hostname;

    if (!TOGETHER_HOSTS.test(host)) {
      return realFetch(url, options);
    }

    stubState.calls += 1;

    // Space out AI completions so `createdAt`-sorted collections
    // (aiLogs) have deterministic ordering.
    await new Promise((resolve) => setTimeout(resolve, 5));

    if (stubState.mode === 'http-error') {
      return jsonResponse(500, {
        error: { message: 'Stub provider failure', type: 'server_error' },
      });
    }

    const requestBody = JSON.parse(options.body);

    if (stubState.mode === 'invalid-json') {
      return jsonResponse(200, {
        id: 'chatcmpl-stub',
        object: 'chat.completion',
        created: 0,
        model: requestBody.model ?? 'stub-model',
        choices: [
          {
            index: 0,
            message: { role: 'assistant', content: 'this is not json' },
            finish_reason: 'stop',
          },
        ],
        usage: { prompt_tokens: 1, completion_tokens: 1, total_tokens: 2 },
      });
    }

    if (stubState.mode === 'empty') {
      return jsonResponse(200, {
        id: 'chatcmpl-stub',
        object: 'chat.completion',
        created: 0,
        model: requestBody.model ?? 'stub-model',
        choices: [
          {
            index: 0,
            message: { role: 'assistant', content: null },
            finish_reason: 'stop',
          },
        ],
        usage: { prompt_tokens: 1, completion_tokens: 1, total_tokens: 2 },
      });
    }

    return jsonResponse(200, buildCompletion(requestBody));
  };

  return realFetch;
}
