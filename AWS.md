To configure your AWS CLI, obtain key from
IAM > Users> Security Credentials TAB > Access keys > Create Access Key > Command Line Interface > 
Select CLI option, check the checkbox accepting terms and click 'submit' to check the credentails.

-----------------------------------------------------------------------------------

If you have a private key and would like to generate a public key from it, you can run this command on linux terminal:

ssh-keygen -y -f PrivateKEYPAIR.pem

In aws console, EC2 -> network & Security -> keypairs, select, Actions-> Import keypair and provide the generated publickey to reuse the same private key across accounts in different Availability zones.
-----------------------------------------------------------------------------------

To quickly install nginx on the server ---->
sudo yum install nginx

and to start it--->
sudo sytemctl start nginx

-----------------------------------------------------------------------
Commands to move aws key pair between instances.
scp -i ec2-user.pem ec2-user.pem ec2-user@PublicIPAddress
-----------------------------------------------------------------------------------

  After creating a elastic block storage and attaching it to an Ec2 instance, if you would like to check if its available on that instance run command on the terminal
  --> lsblk 
  Then verify if there is a file system on the storage by using 
  --> sudo file -s /dev/xvdf
  If there is a file system, create a folder and mount it using
  --> sudo mkdir /ebsdemo
  --> sudo mount /dev/xvdf /ebsdemo
  All the data in that block storage should be now in 'ebsdemo' folder. Verify using --> cd /ebsdemo/

  If you dont have a file system on the directory(like when you run sudo file -s /dev/xvdf, you see :data in the outpu), then follow steps as below.
  Create a file system --> sudo mkfs -t xfs /dev/xvdf
  now you should see the XFS file system on the block storage.
  go to root and create ebsdemo folder.. --> cd /
  --> sudo mkdir /ebsdemo
  Mount it to the directory ebsdemo.
  --> sudo mount /dev/xvdf /ebsdemo
  
  To verify if the file system is mounted with -> df -k (/dev/xvdf should show in the list)

** remember this is temporary mount to the server. This is removed after a restart.
If you want the volume to be persistent, modify the system file. Steps below
--> Find the unique id for the ebsvolume using --> sudo blkid (for /dev/xvdf)
--> modify the fstab in etc folder --> sudo vi /etc/fstab
append the UUID of the new ebsvolume. like below
--> UUID=***UUID caputed in step above*** /ebsdemo xfs defaults,nofail





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
-------------------------------------------------------------------
IAM Permissions
 - Users or groups can be assigned json documents called 'IAM Policies'
 - Policies define permissions of the user.
 - 
