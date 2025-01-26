# EpubGemini
> Simple script for applying a prompt to all chapters of an epub and storing the result as a the new chapter.


## Overview
The script runs makes an api call to a gemini model for each chapter. 
This can be used to edit grammer, improve flow/clarity, etc. 
It really just depeneds on what you can get the llm to do. 
The program was mainly designed around the free teir of the api so it follows the requests per minute and tokens per minute (there is a 1500 requests per day cap but i didnt implement it ¯\_(ツ)_/¯).
Since we are directly pasting the response into a .html file you really need to beg gemini to not include random stuff like "Ok, here is your chapter that has been....". It isnt really important since 
most html renders are pretty lenient about compile errors but test your prompts.



## Usage 


| flag        | required  | defualt vale  | defualt value                                                                                                         | 
|-------------|-----------|---------------|-----------------------------------------------------------------------------------------------------------------------|
| -f           | required  | N/A           | input .epub file (only .epub)                                                                                         |   
| -d           | optional  | "output"      | output directory (this is where each chapter is stored before it is combined into an epub, not useful currently)      |   
| -cb          | optional  | 0             | Number of predceeding chapters included as context for each prompt                                                    |   
| -ca          | optional  | 0             | Same as cb but chapters after                                                                                         |   
| -key         | required  | N/A           | Your Gemini API key                                                                                                   |   
| -prompt      | required  | N/A           | This is the instruction for each chapter                                                                              |   
| -instruction | required  | N/A           | The instruction provided to at the begining (You are an expert...)                                                    |   
| -model       | required  | N/A           | The gemini model used                                                                                                 |   
| -j           | optional  | N/A           | You can create a json file with all of the above obptions for convenice (dont include any other flags if you use -j)  |   

### Example

Just use the -j, idk why I inclueded the cli option.

```cmd
EpubGemini.exe -j test.json
```

test.json 


```json 
{
  "file": "inut.epub",
  "directory": "output",
  "contextBefore": 10,
  "contextAfter": 10,
  "APIKey": "",
  "prompt": "Process the following HTML while maintaining valid syntax:\n\n",
  "instruction": "You are an expert text editor. \n\t\tYou specialize in web novels (Fantasy and Cultivation).\n\n\t\tYour task is to:\n        1. Fix grammatical mistakes\n        2. Remove text artifacts not part of the original content\n        3. Maintain consistency in character details\n        4. Improve clarity if necessary (prefer smaller edits)\n\n        Always preserve the essential meaning and structure of the original text.\n\t\tDo not add any formating or extra content to response.\n\t\tDo not include additional context in the response",
  "model": "gemini-1.5-flash"
}
```
