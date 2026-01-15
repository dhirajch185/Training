Where to build your servers?
Compliance - like GDPR
Proximity - To reduce lag
Services - Some features or services are not available in all regions.
Pricing - Vary from region to region.

Availability zones - going into the region. Isolated from disasters. High bandwidth and low latency.

AWS POints of presence (Edge locations) - Deliver content to end users with low latency.

AWS has Global Services:
• Identity and Access Management (IAM)
• Route 53 (DNS service)
• CloudFront (Content Delivery Network)
• WAF (Web Application Firewall)

 Most AWS services are Region-scoped:
• Amazon EC2 (Infrastructure as a Service)
• Elastic Beanstalk (Platform as a Service)
• Lambda (Function as a Service)
• Rekognition (Software as a Service)

Services available per region table - https://aws.amazon.com/about-aws/global-infrastructure/regional-product-services/

---------------------------------------------------------------------------
IAM is global service
  - Root account is created by default.
  - users are people within an organization and can be grouped
  - Groups can only contain users (like 'Developers group', 'testers group', 'Operations team' etc)
  - Users can belong to multiple groups.
  - they dont need to be in any group either.
