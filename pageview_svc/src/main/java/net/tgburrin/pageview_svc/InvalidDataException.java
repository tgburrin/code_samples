package net.tgburrin.pageview_svc;

import org.springframework.http.HttpStatus;
import org.springframework.web.bind.annotation.ResponseStatus;

@SuppressWarnings("serial")
@ResponseStatus(value = HttpStatus.BAD_REQUEST)
public class InvalidDataException extends Exception {
	public InvalidDataException(String message) {
		super(message);
	}

	public InvalidDataException(String message, Throwable err) {
		super(message, err);
	}
}
