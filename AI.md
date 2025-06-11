Local llm

Step1 : install python 3 and ollama service on the machine.
step2: python3 -m venv venv --> Next navigate to scripts folder in venv and activate
step3: pip install ollama
step4: write python code to send the prompt like below

---------------------------------------- ollama library based example ---------------------------------
import ollama

PROMPT = """
ONLY Generate an ideal Dockerfile for {language} with best practices. Do not provide any description
Include:
- Base image
- Installing dependencies
- Setting working directory
- Adding source code
- Running the application
- Multi stage docker build
"""

def generate_dockerfile(language):
    response = ollama.chat(model='llama3.2:1b', messages=[{'role': 'user', 'content': PROMPT.format(language=language)}])
    return response['message']['content']

if __name__ == '__main__':
    language = input("Enter the programming language: ")
    dockerfile = generate_dockerfile(language)
    print("\nGenerated Dockerfile:\n")
    print(dockerfile)

---------------------------------- with gemini model ---------------------------------

import google.generativeai as genai
import os

# Set your API key here
os.environ["GOOGLE_API_KEY"] = "****************My Google API Key*************"

# Configure the Gemini model
genai.configure(api_key=os.getenv("GOOGLE_API_KEY"))
model = genai.GenerativeModel('gemini-1.5-pro')

PROMPT = """
Generate an ideal Dockerfile for {language} with best practices. Just share the dockerfile without any explanation between two lines to make copying dockerfile easy.
Include:
- Base image
- Installing dependencies
- Setting working directory
- Adding source code
- Running the application
- Multi stage docker build
"""

def generate_dockerfile(language):
    response = model.generate_content(PROMPT.format(language=language))
    return response.text

if __name__ == '__main__':
    language = input("Enter the programming language: ")
    dockerfile = generate_dockerfile(language)
    print("\nGenerated Dockerfile:\n")
    print(dockerfile)

-------------------------------------------------------------------------------------------------

Gen AI

- Computing power access made AI possible and accessible to most of us.
What’s the difference between AI, ML and gen AI?

** AI - Building machines or systems that can do tasks what requires humans for it. like learning, problem solving, decision making.

** ML - A subset AI. Its all about using data to train machines to perform specific tasks. AI systems uses Math ( A system of euqtions) to solve these problems in ML models. Based on data we input, we get different output. So data plays a huge role.

** GenAI (Generative AI) - Its a subset of ML. It focuses on creating new content like images, audio, text. It generates someting entirely new. It learns underlying patterns and structures of the data they are trained on. Use what they learned and genreate new content. To learn these patterns well, these need a lot of data.

Machine learning models predict the future based on existing data, much like how humans use experience to make educated guesses. However, while humans might rely on intuition or gut feelings, these models use probability.
Data can be - Numbers (temperature, date), Text description (description of an item's appearance),Images or sounds. Depending on what you want your ML model to do, some data is relevant and some is irrelevant.

Data Quality depends on --> Accuracy, Size of the dataset, Representative (Data needs to be representative and inclusive, otherwise it can lead to skewed samples and biased outcomes. If a dataset about customer preferences is missing information about a certain demographic, the model might make inaccurate or biased generalizations about that group.), consistency (  Inconsistent data formats or labeling can confuse the model and hinder its ability to learn effectively. Imagine trying to assemble a puzzle where some pieces are labeled with numbers and others with letters—it would be a mess.), Relevance ( The data must be relevant to the task the AI is designed to perform. For example, data about traffic patterns in London is unlikely to be helpful for predicting crop yields in Banglore.

Data Accessibility - The ability of AI systems to effectively utilize this data is directly tied to data accessibility 
        ** Availability - If the necessary data simply isn't available, the AI model can't be trained. For some problems, the data might exist, but it might be locked behind paywalls or restricted due to privacy concerns.
        ** Cost - Gathering and cleaning data can be expensive. The cost of acquiring high-quality data can be a significant barrier to AI development, especially for smaller organizations.
    ** Format - Data needs to be in a format that the AI model can understand and process. Converting data into the appropriate format can be time-consuming and technically challenging.

Types of Data: Structured (organized in tables, mostly from a relational DB) and unstructured (pdf, emails, social media posts, images,audio and video).

Machine learning - 3 primary learning approaches; each with its own data requirements
                                                ---> supervised - Supervised models rely on labeled data
                                                    unsupervised - unsupervised models work with unlabeled data
                                                    reinforcement learning - reinforcement learning learns through interaction and feedback
Labeled Data - An image dataset for training a cat-detection model would label each picture as either a cat or dog. Similarly, a set of customer reviews might be labeled as positive, negative, or neutral. These labels enable algorithms to learn relationships and make accurate predictions.

Unlabeled Data - A collection of unorganized photos, a stream of audio recordings, or website traffic logs without user categorization. In these cases, algorithms must independently discover patterns and structures within the data, as there are no pre-existing labels to guide the learning process.

Supervised Machine Learning - Train models on labeled data, where each input is paired with its correct output, allowing the model to learn the relationship between them. The model's goal is to identify patterns and relationships within this labeled data, enabling it to accurately predict outputs for new, unseen inputs.
Example------->Predicting housing prices is a common example of supervised learning. A model is trained on a dataset where each house has labeled data, such as its size, number of bedrooms, location, and the corresponding sale price. This labeled data allows the algorithm to learn the relationship between the features of a house and its price. Once trained, the model can then predict the price of a new house based on its features.

Unsupervised Machine Learning - ML model deal with raw, unlabeled data to find natural groupings. Instead of learning from labeled data, it dives headfirst into a sea of unlabeled data.
Example-------> An unsupervised learning algorithm could analyze customer purchase history from a company's database. It might uncover hidden segments of customers with similar buying habits, even though you never explicitly labeled those segments beforehand. This can be incredibly valuable for targeted marketing or product recommendations.

Think of it as exploratory analysis. Unsupervised learning helps you understand the underlying structure of your data and uncover insights that you might not have even known to look for.

Reinforcement learning  --> learning through interaction and feedback. Imagine a robot learning to navigate a maze. It starts with no knowledge of the maze's layout. As it explores and interacts with the maze, it collects data—bumping into walls (negative feedback) or finding shortcuts (positive feedback). Through this process of trial and error, the algorithm learns which actions lead to the best outcomes. 
In reinforcement learning, the algorithm learns to maximize rewards and minimize penalties by interacting with its environment.
This type of learning is particularly useful in situations where you can't provide explicit instructions or labeled data. For example, you could use reinforcement learning to train a self-driving car to navigate complex traffic situations or to optimize the performance of a robot in a manufacturing plant.

------------->
Gather data by ingestion --> Pub/Sub handles real-time streaming data processing, regardless of the structure of the data. 
                            Cloud Storage is well-suited for storing unstructured data.
                            Cloud SQL and Cloud Spanner are used to manage structured data.

Data preparation -----> Its the process of cleaning and transforming raw data into a usable format for analysis or model training. This involves formatting and labeling data properly. 
                        Google Cloud offers tools like BigQuery for data analysis and Data Catalog for data governance. These tools help prepare data for ML models. 
                        With BigQuery, you can filter data, correct its inconsistencies, and handle missing values. 
                        With Data Catalog, you can find relevant data for your ML projects. This tool provides a centralized repository to easily discover datasets in your organization.

Train your Model -----> The process of creating your ML model using data is called model training. Google Cloud's Vertex AI platform provides a managed environment for training ML models. 
                        With Vertex AI, you can set parameters and build your model, using prebuilt containers for popular machine learning frameworks, custom training jobs, and tools for model evaluation.
                        Vertex AI also provides access to powerful computing resources to make the model training process faster.

Deploy & Predict ----> Model deployment is the process of making a trained model available for use. 

                        Vertex AI simplifies this by providing tools to put the model into action for generating predictions. This includes scaling the deployment, which means adjusting the resources allocated to the model to handle varying amounts of usage.

Model Management ----> Model management is the process of managing and maintaining your models over time. Google Cloud offers tools for managing the entire lifecycle of ML models. This includes the following:

                        Versioning: Keep track of different versions of the model.
                        Performance tracking: Review the model metrics to check the model's performance.
                        Drift monitoring: Watch for changes in the model's accuracy over time.
                        Data management: Use Vertex AI Feature Store to manage the data features the model uses.
                        Storage: Use Vertex AI Model Garden to store and organize the models in one place.
                        Automate: Use Vertex AI Pipelines to automate your machine learning tasks.


------------------------------------------------------

Deep Learning - A powerful subset of machine learning, distinguished by its use of artificial neural networks. These networks enable the processing of highly complex patterns and the generation of sophisticated predictions.

Neural networks can leverage both labeled and unlabeled data, a strategy known as semi-supervised learning. They train on a blend of a small amount of labeled data and a large amount of unlabeled data. That way, they learn foundational concepts and generalize effectively to novel (new, not previously experienced)  examples.

Large language models (LLMs)​​​

One particularly exciting type of foundation model is the LLM. These models are specifically designed to understand and generate human language. 

They can translate languages, write different kinds of creative content, and answer your questions in an informative way, even if they are open ended, challenging, or strange. This is likely the most common foundation model you've encountered, such as in popular generative AI chatbots like Gemini. They also help power many search engines you use today.

Diffusion models
Diffusion models are another type of foundational model. They excel in generating high-quality images, audio, and even video by iteratively refining noise (or unstructured/random data and patterns) into structured data.

To summarize, deep learning provides the core technology, foundation models are the powerful architectures built on deep learning, and generative AI is the application of these models to create new and original content.


--------------------------------------------------------------

Modality -  Modality refers to the type of data the model can process and generate, such as text, images, video, or audio. If your application focuses on a single data type, like generating text-based articles or creating audio files, you'll want to choose a model optimized for that specific modality.

context window - The amount of information a model can consider at one time when generating a response. A larger context window allows the model to "remember" more of the conversation or document, leading to more coherent and relevant outputs, especially for longer texts or complex tasks.  larger context windows often come with increased computational costs. You need to balance the need for context with the practical limitations of your resources.

Security -  Consider the model's security features, including data encryption, access controls, and vulnerability management. Ensure the model complies with relevant security standards and regulations for your industry.

Availability and reliability - Choose a model that is consistently available and performs reliably under load. Consider factors like uptime guarantees, redundancy, and disaster recovery mechanisms.

Cost - Generative AI models can vary significantly in cost. Consider the pricing model, which might be based on usage, compute time, or other factors. Evaluate the cost-effectiveness of the model in relation to your budget and the expected value of your application. This is where selecting the right model for the right task is important. Be sure to match the model to the task; bigger isn't always better, and multi-modal capabilities aren't always necessary.

Performance - The performance of the model, including its accuracy, speed, and efficiency, is a critical factor. Evaluate the model's performance on relevant benchmarks and datasets. Consider the trade-offs between performance and cost.

Fine-tuning & customization -  Some models can be fine-tuned or customized for specific tasks or domains. If you have a specialized use case, consider models that offer fine-tuning capabilities. This often involves training the model further on a specific dataset related to your use case.

 Ease of integration  - The ease of integrating the model into your existing systems and workflows is an important consideration. Look for models that offer well-documented APIs and SDKs.

With Vertex AI you can access models developed by Google including Gemini, Gemma, Imagen, and Veo. You can also access proprietary third-party models, and openly available models.

Gemini -  Gemini, a multimodal model, can understand and operate across diverse data formats, such as text, images, audio, and video. Gemini's multimodal design supports applications that require complex multimodal understanding, advanced conversational AI, content creation, and nuanced question answering.

Gemma - A family of lightweight, open models is built upon the research and technology behind Gemini. They offer developers a user-friendly and customizable solution for local deployments and specialized AI applications.

Imagen - A powerful text-to-image diffusion model, it excels at generating high-quality images from textual descriptions. This makes it invaluable for creative design, ecommerce visualization, and content creation.

Veo - A model capable of generating video content. It can produce videos based on textual descriptions or still images. Its functionality allows for the creation of moving images for applications such as film production, advertising, and online content.

------------------------------

Foundation model limitations - 
    Data dependency - The performance of foundation models is heavily data-dependent. They require large datasets, and any biases or incompleteness in that data will inevitably seep into their outputs. It's like asking a student to write an essay on a book they haven't read. If the data or questions are inaccurate or biased, the AI's performance will suffer.

Knowledge cutoff - Knowledge cutoff is the last date that an AI model was trained on new information. Models with older knowledge cutoffs may not know about recent events or discoveries. This can lead to incorrect or outdated answers, since AI models don't automatically update with the latest happenings around the world. For example, if an AI tool's last training date was in 2022, it wouldn't be able to provide information about events or information that happened after 2022.

Bias - An LLM learns from large amounts of data, which may contain biases. You can think of bias as an unbalanced dataset in LLMs. Due to their statistical learning nature, they can sometimes amplify existing biases present in the data. Even subtle biases in the training data can be magnified in the model's outputs.

Fairness - Even with perfectly balanced data, defining what constitutes fairness in an LLM's output is a complex task. Fairness can be interpreted in various ways. Fairness assessments for generative AI models, while valuable, have inherent limitations. These evaluations typically focus on specific categories of bias, potentially overlooking other forms of prejudice. Consequently, these benchmarks do not provide a complete picture of all potential risks associated with the models' outputs, highlighting the ongoing challenge of achieving truly equitable AI.

Hallucinations - Foundation models can sometimes experience hallucinations, which means they produce outputs that aren't accurate or based on real information. Because foundation models can't verify information against external sources, they may generate factually incorrect or nonsensical responses. These cause significant concern in accuracy-critical applications. The responses might sound convincing, but they are completely wrong. We will cover this in more detail below.

Edge cases - Rare and atypical scenarios can expose a model's weaknesses, leading to errors, misinterpretations, and unexpected results. 

----

Gounding - Tying AI models' answers to specific sources. This makes AI's output more accurate and relevant. Training AI on company's specific data is a good example of gounding in enterprise context. It provides relevant output based on trained company documements and data. Fine tuning and RAG are used for gounding GenAI data.

Generative AI models are amazing at creating content, but sometimes they hallucinate. Grounding is the process of connecting the AI's output to verifiable sources of information—like giving AI a reality check. By providing the model with access to specific data sources, we tether its output to real-world information, reducing the risk of invented content. 

Grounding is essential for building trustworthy and reliable AI applications. By connecting your models to verifiable data, you ensure accuracy and build confidence. It offers several key benefits, including reducing hallucinations, which prevents the AI from generating false or fictional information. Grounding also anchors responses, ensuring the AI's answers are rooted in your provided data sources. Furthermore, it builds trust by enhancing the trustworthiness of the AI's output by providing citations and confidence scores, allowing you to verify the information.


Retrieval-augmented generation (RAG) - There are many different options on how you can ground in data. For example, you can ground in enterprise data or you can ground using Google Search. One common grounding method to do this is with retrieval-augmented generation, or RAG.

RAG is a grounding method that uses search to find relevant information from a knowledge base and provides that information to the LLM, giving it necessary context.

The first step is retrieval. When you ask an AI a question, RAG uses a search engine to find relevant information. This search engine uses an index that understands the semantic meaning of the text, not just keywords. This means it finds information based on meaning, ensuring higher relevance. 
The retrieved information is then added to the prompt given to the AI. This is the augmentation phase. 
The AI then uses this augmented prompt, along with its existing knowledge, to generate a response. This is referred to as the generation phase.

Prompt engineering - Prompting offers the most rapid and straightforward approach to supplying supplementary background information to models. This involves crafting precise prompts to guide the model towards desired outputs. It refines results by understanding the factors that influence a model's responses. However, prompting is limited by the model's existing knowledge; it can't conjure information it hasn't learned.


Fine-tuning - When prompt engineering doesn't deliver the desired outcomes, fine-tuning can enhance your model's performance. Pre-trained models are powerful, but they're designed for general purposes. Tuning helps them excel in specific areas. This process is particularly useful for specific tasks or when you need to enforce specific output formats, especially if you have examples of the desired output.
Tuning involves further training a pre-trained or foundation model on a new dataset specific to your task. This process adjusts the model's parameters, making it more specialized for your needs.

Here are some examples of how tuning can be used:
Fine-tuning a language model to generate creative content in a specific style.
Fine-tuning a code generation model to generate code in a particular programming language.
Fine-tuning a translation model to translate between specific languages or domains.

Humans in the loop (HITL) - Beyond these techniques, we must remember the invaluable role of humans in the loop (HITL). Machine learning models are powerful, but they sometimes need a human touch. For tasks requiring judgment, context, or handling incomplete data, human expertise is essential. HITL systems integrate human input and feedback directly into the ML process. This collaboration makes models more adaptable, especially in areas like the following.

Content moderation - HITL ensures accurate and contextually appropriate moderation of user-generated content, filtering out harmful or inappropriate material that algorithms alone might miss.

Sensitive applications - In fields like healthcare or finance, HITL provides oversight for critical decisions, ensuring accuracy and mitigating risks associated with automated systems.

High-risk decision making - When ML models inform decisions with significant consequences, such as medical diagnoses or criminal justice assessments, HITL acts as a safeguard, providing a layer of human review and accountability.

Pre-generation review - Before deploying ML-generated content or decisions, human experts can review and validate the outputs, catching potential errors or biases before they impact users.

Post-generation review - After ML outputs are deployed, continuous human review and feedback help identify areas for improvement, enabling models to adapt to evolving contexts and user needs.

----------------------------------------------------

Securing AI - Access controls must be implemented to restrict who can access the data. They are also necessary to restrict who can add or input to the data. This is crucial to prevent data poisoning, a malicious attack where bad actors corrupt your AI model’s training data by introducing manipulated data. This can cause your model to learn incorrect patterns and make flawed (biased, inaccurate, or even harmful) predictions. It's akin to someone secretly swapping out vital ingredients in a recipe, leading to a disastrous final dish.

Data prep - Access controls must be implemented to restrict who can access the data. They are also necessary to restrict who can add or input to the data. This is crucial to prevent data poisoning, a malicious attack where bad actors corrupt your AI model’s training data by introducing manipulated data. This can cause your model to learn incorrect patterns and make flawed (biased, inaccurate, or even harmful) predictions. It's akin to someone secretly swapping out vital ingredients in a recipe, leading to a disastrous final dish.

Model Training - s safeguarding both the training data and model parameters from unauthorized access or modification. Model theft, where attackers steal proprietary or sensitive AI models, is a significant threat. Beyond gaining a competitive advantage, the stolen models could be used for malicious purposes, exploiting vulnerabilities or replicating sensitive functionality.

Deploy & Predict - Once your model is trained and ready for action, it's crucial to safeguard it within its operating environment. This means controlling access to the model, determining who can interact with it and how. If you're using a pre-built model, it's essential to verify its source and check for any potential vulnerabilities.

Model management - Continuous monitoring and maintenance of the model's security are essential. Stay up to date on the latest security best practices in the field, specifically for your platform and model, and regularly update to patch vulnerabilities. Monitor model performance and outputs for anomalies or signs of tampering, and regularly review access permissions.

Applying the Secure AI Framework (SAIF) - Google has developed the Secure AI Framework, or SAIF, to establish security standards for building and deploying AI systems responsibly. This comprehensive approach to AI/ML model risk management addresses the key concerns of security professionals in the rapidly evolving landscape of AI. Following the security practices outlined in SAIF can help your organization find and stop threats, automatically strengthen its defenses, and manage the unique risks of each AI system. 

AI offers significant benefits, but it also introduces security risks like data poisoning, model theft, and prompt injection. To address these challenges, a secure foundation for AI applications is essential. Google Cloud's SAIF framework, combined with security tools, can help as you build and maintain secure AI systems.

Responsible AI
Transparency - Users need to understand how their information is being used and how the AI system works. This transparency should extend to all aspects of the AI's operation, including data handling, decision-making processes, and potential biases.

Privacy - Protecting privacy often involves anonymizing or pseudonymizing data, ensuring individuals can't be easily identified. AI models can sometimes inadvertently leak sensitive information from their training data, so it's crucial to implement safeguards to prevent this.

Data quality, bias, and fairness
 Ethical AI requires high quality data
Machine learning and generative AI applications are fundamentally based on data. Therefore, responsible AI requires high-quality data. Inaccurate or incomplete data can lead to biased outcomes.

Understanding and mitigating bias​ - AI systems are not independent of the world they are built in. They can inherit and amplify existing societal biases. This can lead to unfair outcomes, such as a resume-screening tool that favors a certain demographic of candidates due to historical biases in hiring data.

Fairness requires accountability​
Fairness requires accountability. We need to know who is responsible for the AI's actions and understand how it makes decisions. This is where explainability comes in. Explainable AI makes the decision-making processes of AI models transparent and understandable. This is crucial for building trust, debugging errors, and uncovering hidden biases.

Legal implications - Beyond considerations like fairness and bias, AI development is increasingly governed by legal frameworks. Key areas include data privacy, non-discrimination, intellectual property, and product liability. 

-----------------------

Think of generative AI as a powerful engine. The infrastructure is its foundation, the models are its core, and the platform connects everything together. Agents act as the drivers, making decisions and taking action, while applications are the vehicles that take us to our destination.


5 Layers of Gen AI
* Infrastructure - This is the foundation where everything else rests providing computing resources needed for GenAI. Like servers, GPU, TPU and software for storing and running AI models and training data.
* Models - Its the brain of the agent. These are algorithms trained on huge data. They learn patterns and relations in data. This allows to generate new data, translate languages, answer questions etc.  
* Platform - its between agents and models to provide infra to interact. Provides necessary tools and infra for AI development.Offers API, data management capabilities and model deployment tools. It acts as backbone of the system and more like a bridge.
* Agents - Software that learns how to best achieve a goal based on inputs and tools available to it. This focus on autonomous action (independently set goals and carry them out within a defined environment). Agents use multiple tools, analyze situations and make informed decisions without constant human inputs. They perfom multi-step taks what a model alone cannot. Like researching a topic, troubleshooting code or accessing a system by chaining actions together. You can have 'customer agents', 'code agents', 'data agents' and more... 
* GenAI applications - User facting applications. Users interact with and leverage the capabilities of AI. Examples- Gemini app, NotebookLM


Agents: Think of agents as the intelligent pieces within a larger gen AI powered application. They bring specific capabilities to the table.

Understanding and responding to natural language - This allows applications to have more intuitive interfaces, like chatbots that can actually understand complex requests.
Automating complex tasks - Agents can handle multi-step processes within an application, such as gathering information, making decisions based on that information, and taking action.
Personalization - Agents can learn user preferences and tailor the application experience accordingly.

Example: A personalized learning app - An agent could assess a student's knowledge, recommend relevant learning materials, and even generate personalized exercises. The application would provide the structure for lessons and track progress.

 ***Gen AI agents will mostly fall under two categories: conversational agents or workflow agents.

Conversational agents - Conversational agents are designed to understand what you mean, not just what you say, and respond in a way that makes sense. 
You provide input: You can type a message or speak to the agent.
The agent understands: Using powerful AI, it figures out the meaning and intention behind your words.
The agent calls a tool: Based on your request, the agent might need to gather additional information or perform an action. This could involve searching the web, accessing a database, or interacting with another software application.
The agent generates a response: It formulates an answer that's relevant to your request and sounds natural.
The agent delivers the response: You'll see or hear the answer depending on how you interacted with it.

Workflow agents - Workflow agents are designed to streamline your work and make sure things get done efficiently and correctly by automating tasks or going through complex processes.
Workflow agents are designed to streamline your work and make sure things get done efficiently and correctly by automating tasks or going through complex processes.



You provide input: You define a task or trigger a process like submitting a form, uploading a file, initiating a scheduled event, or even ordering a product online.
The agent understands: The agent is the software that automates those steps. It interprets the task's requirements and defines the series of steps needed to complete the task.
The agent calls a tool: Based on the workflow's definition, the agent executes a series of actions. This could involve data transformation, file transfer, sending notifications, integrating with external systems, or initiating other automated processes using APIs.
The agent generates a result/output: It compiles the outcome of the executed actions, which might be a report, a data file, a confirmation message, or an updated status within a system.
The agent delivers the result/output: The agent delivers the output to the designated recipient(s) or system(s), such as via email, a dashboard, a database update, or a file storage location.

Advanced prompt engineering frameworks - The reasoning loop often utilizes advanced prompt engineering frameworks to guide its decision-making process. These frameworks can include:

Simple rule-based calculations
Complex thought chains
Machine learning algorithms
Probabilistic reasoning techniques

Examples of such frameworks include ReAct or chain-of-thought (CoT) prompting.







