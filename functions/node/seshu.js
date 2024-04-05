// seshu is japanese meaning "to consume"
// this is a function consumes the content of a target URL and returns
// an array of objects representing potential events on the target page

import dotenv from 'dotenv';
dotenv.config();
import fs from 'fs';
// import { TurndownService } from 'turndown';
import { NodeHtmlMarkdown, NodeHtmlMarkdownOptions } from 'node-html-markdown';
import { ZenRows } from 'zenrows';
import { OpenAI } from 'openai';

// TODO: dropd `turndown` => `node-html-markdown` https://github.com/crosstype/node-html-markdown#readme

const zrClient = new ZenRows(process.env.ZENROWS_API_KEY);

const nhm = new NodeHtmlMarkdown(
  /* options (optional) */ {},
  /* customTransformers (optional) */ undefined,
  /* customCodeBlockTranslators (optional) */ undefined,
);

export default async function handler(event, context, callback) {
  console.log('~hello!');
  return 'Hello, World!';
}

const _handler = async (event, context, callback) => {
  // Create an instance of OpenAI using your key
  const openai = new OpenAI({
    apiKey: process.env.OPENAI_API_KEY,
  });

  // zernrows parse fetch
  const url =
    'https://www.meetup.com/find/?location=us--co--Cortez&source=EVENTS&distance=hundredMiles';

  let _data;
  try {
    const { data } = await zrClient.get(url, {
      js_render: 'true',
      wait: '2500',
      // we might want html mode instead, for markdown parsing
      // autoparse: 'true',
    });
    _data = data;
  } catch (error) {
    console.error(error.message);
    if (error.response) {
      console.error(error.response.data);
    }
  }

  const turndownService = new TurndownService();
  turndownService.keep(['body']);
  turndownService.remove(['script', 'style']);

  try {
    const markdown = turndownService.turndown(_data.toString());
    const arrOfLines = markdown
      .replace(/\n\s*\n/g, '\n')
      ?.split('\n')
      .map((line) => line.trim())
      .filter((line) => line.length > 0);

    const config = {
      messages: [
        {
          role: 'user',
          content: `You are a helpful LLM capable of accepting an array of strings and reorganizing them according to patterns only an LLM is capable of recognizing.

  Your goal is to take the javascript array input I will provide, called the \`textStrings\` below and return a grouped array of objects. Each object should represent a single event, where it's keys are the event metadata associated with the categories below that are to be searched for. There should be no duplicate keys. Each object consists of no more than one of a given event metadata. When forming these groups, prioritize proximity (meaning, the closer two strings are in array position) in creating the event objects in the returned array of objects. In other words, the closer two strings are together, the higher the likelihood that they are two different event metadata items for the same event.

  Do not provide me with example code to achieve this task. Only an LLM (you are an LLM) is capable of reading the array of text strings and determining which string is a relevance match for which category can resolve this task. Javascript alone cannot resolve this query.

  Do not explain how code might be used to achieve this task. Do not explain how regex might accomplish this task. Only an LLM is capable of this pattern matching task. My expectation is a response from you that is an array of objects, where the keys are the event metadata from the categories below.

  Do not return an ordered list of strings. Return an array of objects, where each object is a single event, and the keys of each object are the event metadata from the categories below.

  It is understood that the strings in the input below are in some cases not a categorical match for the event metadata categories below. This is acceptable. The LLM is capable of determining which strings are a relevance match for which category. It is acceptable to discard strings that are not a relevance match for any category.

  The categories to search for relevance matches in are as follows:
  =====
  1. Event title
  2. Event location
  3. Event date
  4. Event URL
  5. Event description

  Note that some keys may be missing, for example, in the example below, the "event description" is missing. This is acceptable. The event metadata keys are not guaranteed to be present in the input array of strings.

  Do not truncate the response with an ellipsis \`...\`, list the full event array in it's entirety. Your response must be a javascript array of \`events\` objects following this example schema:

  \`\`\`
  const events = [{
    event_title: 'Meetup at the park',
    event_location: 'Espanola, NM 87532',
    event_date: 'Sep 26, 5:30-7:30pm',
    event_url: 'http://example.com/events/12345'
  },
  {
    event_title: 'Yoga at sunrise',
    event_location: 'Espanola, NM 87532',
    event_date: 'Oct 13, 6:30-7:30am',
    event_url: 'http://example.com/events/98765'
  }]
  \`\`\`

  The input is:
  =====
  const textStrings = ${JSON.stringify(arrOfLines)}
  `,
        },
      ],
      model: 'gpt-3.5-turbo-16k',
    };

    const chatCompletion = await openai.chat.completions.create(config);

    // fs.writeFileSync(
    //   `${process.cwd()}/scrapes/scrape-llm-${new Date().toISOString()}.html`,
    //   '\n\n\n========== scrape target URL ============\n\n\n\n' +
    //     url +
    //     '\n\n\n========== HTML to markdown result ============\n\n\n\n' +
    //     JSON.stringify(arrOfLines) +
    //     '\n\n\n========== openAI config ============\n\n\n\n' +
    //     JSON.stringify(config) +
    //     '\n\n\n========== response ============\n\n\n\n' +
    //     JSON.stringify(chatCompletion),
    //   'utf8',
    // );
    return {
      statusCode: 200,
      headers: { 'Content-Type': 'text/plain' },
      body: `Hello, World! Response from Open AI: ${JSON.stringify(
        chatCompletion,
      )}`,
    };
  } catch (error) {
    console.error(error.message);
    if (error.response) {
      console.error(error.response.data);
    }
  }
};
