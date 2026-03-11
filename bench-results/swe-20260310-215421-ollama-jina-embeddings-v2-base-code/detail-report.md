# SWE-Bench Detail Report

Generated: 2026-03-10 20:57 UTC

---

## java-hard [java]

**Issue:** NullPointerException in W3CDom when a custom document is passed

> When providing a custom output Document to W3CDom#convert, a NullPointerException is thrown in the W3CBuilder constructor.
> 
> The W3CBuilder constructor retrieves the context element via doc.getUserData(ContextProperty). When converting with a custom output Document (e.g., one created via DocumentBuilderFactory.newDocumentBuilder().newDocument()), the context element is null because no context property has been set on the custom document. The code then unconditionally calls contextElement.ownerDocument() without checking for null first, causing the NPE.
> 
> Stacktrace:
> ```
> java.lang.NullPointerException: Cannot invoke "org.jsoup.nodes.Element.ownerDocument()" because "this.contextElement" is null
> 
> 	at org.jsoup.helper.W3CDom$W3CBuilder.<init>(W3CDom.java:364)
> 	at org.jsoup.helper.W3CDom.convert(W3CDom.java:250)
> 	at org.jsoup.helper.W3CDom.convert(W3CDom.java:238)
> ```
> 
> Reproduction steps:
> 1. Parse an HTML string using Jsoup.parse()
> 2. Create a custom W3C Document via DocumentBuilderFactory.newDocumentBuilder().newDocument()
> 3. Call new W3CDom().convert(jsoupDoc, customDocument)
> 4. Observe the NullPointerException
> 
> Expected behavior: The conversion should succeed when a custom output Document is provided. The contextElement null case should be handled gracefully, as it is a valid scenario when users supply their own output Document.

### Metrics

| Scenario | Duration | Input Tok | Cache Read | Output Tok | Cost |
|----------|----------|-----------|------------|------------|------|
| **baseline** | 28.0s | 7 | 83773 | 1023 | $0.0876 |
| **with-lumen** | 18.9s | 8 | 67252 | 778 | $0.0775 |

### baseline

**Rating: Good**

```diff
diff --git a/src/main/java/org/jsoup/helper/W3CDom.java b/src/main/java/org/jsoup/helper/W3CDom.java
index 6ce0abd..cfd0d78 100644
--- a/src/main/java/org/jsoup/helper/W3CDom.java
+++ b/src/main/java/org/jsoup/helper/W3CDom.java
@@ -360,7 +360,7 @@ public class W3CDom {
             namespacesStack.push(new HashMap<>());
             dest = doc;
             contextElement = (org.jsoup.nodes.Element) doc.getUserData(ContextProperty); // Track the context jsoup Element, so we can save the corresponding w3c element
-            final org.jsoup.nodes.Document inDoc = contextElement.ownerDocument();
+            final org.jsoup.nodes.Document inDoc = contextElement != null ? contextElement.ownerDocument() : null;
             if (namespaceAware && inDoc != null && inDoc.parser().getTreeBuilder() instanceof HtmlTreeBuilder) {
               // as per the WHATWG HTML5 spec § 2.1.3, elements are in the HTML namespace by default
               namespacesStack.peek().put("", Parser.NamespaceHtml);
```

### with-lumen

**Rating: INVALID (lumen not used)**

```diff
diff --git a/src/main/java/org/jsoup/helper/W3CDom.java b/src/main/java/org/jsoup/helper/W3CDom.java
index 6ce0abd..cfd0d78 100644
--- a/src/main/java/org/jsoup/helper/W3CDom.java
+++ b/src/main/java/org/jsoup/helper/W3CDom.java
@@ -360,7 +360,7 @@ public class W3CDom {
             namespacesStack.push(new HashMap<>());
             dest = doc;
             contextElement = (org.jsoup.nodes.Element) doc.getUserData(ContextProperty); // Track the context jsoup Element, so we can save the corresponding w3c element
-            final org.jsoup.nodes.Document inDoc = contextElement.ownerDocument();
+            final org.jsoup.nodes.Document inDoc = contextElement != null ? contextElement.ownerDocument() : null;
             if (namespaceAware && inDoc != null && inDoc.parser().getTreeBuilder() instanceof HtmlTreeBuilder) {
               // as per the WHATWG HTML5 spec § 2.1.3, elements are in the HTML namespace by default
               namespacesStack.peek().put("", Parser.NamespaceHtml);
```


