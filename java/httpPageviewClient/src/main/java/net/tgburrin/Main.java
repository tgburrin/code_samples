package net.tgburrin;

import java.io.IOException;
import java.net.URI;
import java.net.http.HttpClient;
import java.net.http.HttpClient.Version;
import java.net.http.HttpRequest;
import java.net.http.HttpRequest.BodyPublishers;
import java.net.http.HttpResponse;
import java.net.http.HttpResponse.BodyHandlers;
import java.util.ArrayList;
import java.util.concurrent.atomic.AtomicInteger;

import com.google.gson.JsonElement;
import com.google.gson.JsonObject;
import com.google.gson.JsonParser;

public class Main extends Thread {
	private static AtomicInteger msgCounter = new AtomicInteger(0);
	private static int statsTimeSec = 5;

	private static String webHost = "webserver01:8080";
	private static String apiVersion = "v1";
	private static String urlBase = "/" + apiVersion + "/content";

	private static int numberOfClients = 48;
	private static int numberOfViews = 10000;

	private int threadNumber = 0;
	private String contentId = null;

	public Main() {
	}

	public Main(int tn, String cid) {
		threadNumber = tn;
		contentId = cid;
	}

	@Override
	public void run() {
		System.out.println("Starting child " + threadNumber + " (" + contentId + ")");
		HttpClient cli = HttpClient.newBuilder().version(Version.HTTP_2).build();

		for (int i = 0; i < numberOfViews; i++) {
			try {
				HttpRequest req = HttpRequest.newBuilder()
						.uri(URI.create("http://" + webHost + "/" + apiVersion + "/pageview/" + contentId))
						.POST(BodyPublishers.noBody()).build();

				HttpResponse<String> resp = cli.send(req, BodyHandlers.ofString());

				if (resp.statusCode() != 204) {
					System.err.println("Could not post pageview: " + resp.body());
					System.exit(1);
				}

				msgCounter.incrementAndGet();
			} catch (IOException | InterruptedException e) {
				System.err.println("Could not make post request: " + e.getMessage());
				System.exit(1);
			}
		}
	}

	public static ArrayList<String> getContentIds() throws Exception {
		HttpClient cli = HttpClient.newBuilder().version(Version.HTTP_2).build();

		HttpRequest req = HttpRequest.newBuilder().uri(URI.create("http://" + webHost + "/" + urlBase)).GET().build();

		HttpResponse<String> resp = cli.send(req, BodyHandlers.ofString());
		String body = resp.body();

		if (resp.statusCode() != 200)
			throw new Exception("Could not get content list: " + body);

		ArrayList<String> contentIds = new ArrayList<String>();
		JsonObject payload = (new JsonParser().parse(body)).getAsJsonObject();
		if (payload.has("data") && payload.get("data").isJsonArray()) {
			for (JsonElement cid : payload.getAsJsonArray("data"))
				contentIds.add(cid.getAsJsonObject().get("id").getAsString());
		}

		return contentIds;
	}

	public static void main(String[] args) {
		ArrayList<String> contentIds = null;
		try {
			contentIds = getContentIds();
		} catch (Exception e) {
			System.err.println("Could not get content id list: " + e.getMessage());
			System.exit(1);
		}

		if (contentIds != null) {

			ArrayList<Main> children = new ArrayList<Main>();

			Thread thread = new Thread() {
				@Override
				public void run() {
					while (true) {
						try {
							Thread.sleep(statsTimeSec * 1000);
						} catch (InterruptedException e) {
							break;
						}
						int msgCount = msgCounter.getAndSet(0);
						System.out.println(((float) msgCount / (float) statsTimeSec) + " msgs/sec");
					}
				}
			};
			thread.start();

			int cidIdx = 0;
			for (int i = 0; i < numberOfClients; i++) {
				if (cidIdx >= contentIds.size())
					cidIdx = 0;

				Main c = new Main(i + 1, contentIds.get(cidIdx++));
				c.start();
				children.add(c);
			}

			for (Main c : children) {
				try {
					c.join();
				} catch (InterruptedException e) {
					System.err.println("Error from child: " + e.getMessage());
				}
			}

			thread.interrupt();
			try {
				thread.join();
			} catch (InterruptedException e) {
			}

		}

		System.exit(0);
	}
}
